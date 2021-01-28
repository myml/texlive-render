docker-build:
	docker build --target lualatex --tag hub.deepin.com/dstore/lualatex .
	docker build --target texlive-render --tag hub.deepin.com/dstore/texlive-render .
docker-release:
	docker push hub.deepin.com/dstore/lualatex
	docker push hub.deepin.com/dstore/texlive-render