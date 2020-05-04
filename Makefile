BINARY := cake
SHELL := /usr/bin/env bash
export GO111MODULE := on
BIN_DIR := bin
PLATFORMS := windows linux darwin
OSFLAG := $(shell go env GOHOSTOS)
GOPATH := $(shell go env GOPATH)

define STATIK_FILE
package statik

import (
	"github.com/rakyll/statik/fs"
)

func init() {
	data := "not needed for the embedded binary"
	fs.Register(data)
}
endef
export STATIK_FILE

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all-binaries
all-binaries: linux darwin windows ## Compile binaries for all supported platforms (linux, darwin and windows)
.PHONY: linux 
linux: embedded ## Compile the cake binary for linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-linux main.go
	hack/upx-${OSFLAG} bin/cake-linux

.PHONY: darwin
darwin: embedded ## Compile the cake binary for mac
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-darwin main.go
	hack/upx-${OSFLAG} bin/cake-darwin

.PHONY: windows 
windows: embedded ## Compile the cake binary for windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake.exe main.go
	hack/upx-${OSFLAG} bin/cake.exe

.PHONY: cake
cake: embedded ## Compile the cake binary
	GOOS=${OSFLAG} GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-${OSFLAG} main.go
	hack/upx-${OSFLAG} bin/cake-${OSFLAG}

.PHONY: build
build: ## Compile the cake binary and nothing else
	GOOS=${OSFLAG} GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-${OSFLAG} main.go

.PHONY: embedded
embedded: ## Compile the linux cake binary for embedding
	@echo "$$STATIK_FILE" > pkg/statik/statik.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-linux-embedded main.go
	hack/upx-${OSFLAG} bin/cake-linux-embedded
	go get github.com/rakyll/statik
	${GOPATH}/bin/statik -f -src=./bin -dest=pkg -include=cake-linux-embedded

.PHONY: test
test: ## Test with go test
	@echo "$$STATIK_FILE" > pkg/statik/statik.go
	go test -v -covermode=count ./...

.PHONY: clean
clean:  ## Clean up all the go modules
	go clean -modcache -cache

.PHONY: tidy
tidy:  ## Clean up all go modules
	go mod tidy

.PHONY: verify-no-efiles
verify-no-efiles:  ## Validate no e-files exist
	hack/check_for_efiles.sh
