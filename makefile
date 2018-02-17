VERSION := $(shell grep -Eo '(\d\.\d\.\d)(-dev)?' main.go)

.PHONY: build check test

linux: linux_amd64
linux_amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/gitwatch-linux-amd64 github.com/adamdecaf/gitwatch

osx: osx_amd64
osx_amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/gitwatch-osx-amd64 github.com/adamdecaf/gitwatch

dist: build linux osx

deps:
	dep ensure -update

check:
	go vet ./...
	go fmt ./...

test: check dist
	go test ./...

build: check
	go build -o gitwatch github.com/adamdecaf/gitwatch
	@chmod +x gitwatch

docker: dist
	docker build -t gitwatch:$(VERSION) .
