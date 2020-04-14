CLUSTER_ENGINE_BINARY := cluster-engine
OSFLAG := $(shell go env GOHOSTOS)

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'


#####################
# Build and cleanup #
#####################

.PHONY: ${CLUSTER_ENGINE_BINARY}
binary:  ${CLUSTER_ENGINE_BINARY} ## Create CLI binary
${CLUSTER_ENGINE_BINARY}:	
	mkdir -p bin/${CLUSTER_ENGINE_BINARY}
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o bin/${CLUSTER_ENGINE_BINARY}/${CLUSTER_ENGINE_BINARY}-${OSFLAG} cmd/${CLUSTER_ENGINE_BINARY}/${CLUSTER_ENGINE_BINARY}.go

.PHONY: c clean
c: clean
clean:  ## Clean up all the go modules
	go clean -modcache -cache

.PHONY: t tidy
t: tidy
tidy:  ## Clean up all go modules
	go mod tidy