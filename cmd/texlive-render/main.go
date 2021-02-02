package main

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/log"
)

var ROOT = os.Getenv("WORKDIR")

func main() {
	if len(ROOT) == 0 {
		ROOT = "/tmp"
	}
	engine := gin.Default()
	engine.Use(func(ctx *gin.Context) {
		ctx.Next()
		err := ctx.Errors.String()
		fmt.Fprint(ctx.Writer, err)
	})
	engine.GET("/", index())
	engine.GET("/pdf/:name", latex("lualatex", "pdf"))
	engine.POST("/pdf/:name", compressed())
	engine.GET("/pdf/:name/:content", luaLatexV2("lualatex", "pdf"))
	engine.GET("/pdf", luaLatexV2("lualatex", "pdf"))
	engine.Run()
}
func compressed() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var buff bytes.Buffer
		w := gzip.NewWriter(&buff)
		n, err := io.Copy(w, ctx.Request.Body)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
		w.Close()
		log.Printf("gzip compressed %v/%v(%f)", buff.Len(), n, float32(buff.Len())/float32(n)*100)
		ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("./%s/", ctx.Param("name"))+base64.RawURLEncoding.EncodeToString(buff.Bytes()))
	}
}
func luaLatexV2(command string, format string) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var buff bytes.Buffer
		name := ctx.Param("name")
		if len(name) > 0 {
			content, err := base64.RawURLEncoding.DecodeString(ctx.Param("content"))
			if err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
			r, err := gzip.NewReader(bytes.NewReader(content))
			if err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
			_, err = io.Copy(&buff, r)
			if err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
		} else {
			resp, err := http.Get(ctx.Query("url"))
			if err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
			defer resp.Body.Close()
			_, err = io.Copy(&buff, resp.Body)

			if err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
			name = base64.RawURLEncoding.EncodeToString([]byte(ctx.Query("url")))
		}

		path := filepath.Join(ROOT, name)
		infile := filepath.Join(path, name+".tex")
		outfile := filepath.Join(path, name+"."+format)
		os.Mkdir(path, 0766)

		err := ioutil.WriteFile(infile, buff.Bytes(), 0644)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		cmd := exec.CommandContext(ctx, command, "--output-format="+format, "--shell-escape", infile)
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("%w: %s", err, out))
			return
		}
		ctx.File(outfile)
	}
}

func latex(command string, format string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		body := []byte(ctx.Query("body"))
		if len(body) == 0 {
			b, err := ioutil.ReadAll(ctx.Request.Body)
			if err != nil {
				return
			}
			body = b
		}
		b := md5.Sum(body)
		key := hex.EncodeToString(b[:])
		log.Println(key)
		path := filepath.Join(ROOT, key)
		infile := filepath.Join(path, key+".tex")
		outfile := filepath.Join(path, key+"."+format)
		_, err := os.Stat(outfile)
		if err == nil {
			ctx.File(outfile)
			return
		}
		os.Mkdir(path, 0766)
		err = ioutil.WriteFile(infile, body, 0644)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		cmd := exec.CommandContext(ctx, command, "--output-format="+format, "--shell-escape", infile)
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("%w: %s", err, out))
			return
		}
		ctx.File(outfile)
	}
}

func index() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("content-type", "text/html; charset=UTF-8")
		fmt.Fprint(ctx.Writer, `
<form action="/pdf/demo.pdf" method="GET">
  <textarea id="text" name="body">
\documentclass[a4paper]{scrartcl}
\usepackage{xltxtra}
\usepackage{plantuml}

\setmainfont[Mapping=tex-text]{WenQuanYi Micro Hei}
\begin{document}

\subsection{简体中文}
打招呼：

\begin{plantuml}
@startuml
小明 -> 小红: 你好
return
@enduml
\end{plantuml}

\end{document}    
  </textarea>
  <button>提交</button>
</form>
		`)
	}
}
