BINARY := cake
SHELL := /usr/bin/env bash
export GO111MODULE := on
BIN_DIR := bin
PLATFORMS := windows linux darwin
OSFLAG := $(shell go env GOHOSTOS)

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all-binaries
all-binaries: linux darwin windows ## Compile binaries for all supported platforms (linux, darwin and windows)
.PHONY: linux 
linux: ## Compile the cake binary for linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-linux main.go

.PHONY: darwin
darwin: ## Compile the cake binary for mac
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-darwin main.go

.PHONY: windows 
windows: ## Compile the cake binary for windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake.exe main.go

.PHONY: cake
cake: ## Compile the cake binary 
	GOOS=${OSFLAG} GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/cake-${OSFLAG} main.go

.PHONY: clean
clean:  ## Clean up all the go modules
	go clean -modcache -cache

.PHONY: tidy
tidy:  ## Clean up all go modules
	go mod tidy

.PHONY: verify-no-efiles
verify-no-efiles:  ## Clean up all go modules
	hack/check_for_efiles.sh
