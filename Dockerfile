FROM golang:1.14 as builder
COPY ./ /src
RUN cd /src && go build -mod vendor ./cmd/...

FROM debian:latest as lualatex

RUN apt-get update -y
RUN apt-get install -y texlive
RUN apt-get install -y texlive-lang-chinese
RUN apt-get install -y texlive-xetex
RUN apt-get install -y texlive-luatex
RUN apt-get install -y default-jre
RUN apt-get install -y graphviz
RUN apt-get install -y ttf-wqy-microhei

COPY plantuml.jar /opt/plantuml/plantuml.jar
ENV PLANTUML_JAR /opt/plantuml/plantuml.jar

COPY plantuml-lautax/ /usr/share/texlive/texmf-dist/tex/lualatex/plantuml
ENTRYPOINT [ "lualatex" ]

FROM lualatex as texlive-render
COPY --from=builder /src/texlive-render /texlive-render
ENTRYPOINT [ "/texlive-render" ]