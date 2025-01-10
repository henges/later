.PHONY: build builddir

build: builddir
	@rm build/later-linux-x86_64 || true
	@GOOS=linux GOARCH=amd64 go build  -o ./build/later-linux-x86_64 .

builddir:
	@mkdir -p build

deploy:
	./deploy.sh
