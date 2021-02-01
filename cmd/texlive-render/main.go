package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
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
	engine.Run()
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
