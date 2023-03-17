NAME=$(shell echo "flow")
GODIR=$(shell go env GOPATH)
DIR=$(shell pwd)
PACKAGE=github.com/vine-io/flow
GIT_COMMIT=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --abbrev=0 --tags --always --match "v*")
GIT_VERSION=github.com/vine-io/flow/pkg/internal/doc
CGO_ENABLED=0
BUILD_DATE=$(shell date +%s)

generate:
	cd $(GODIR)/src && \
	protoc -I=$(GODIR)/src -I=$(DIR)/vendor --gogo_out=:. --deepcopy_out=:. $(PACKAGE)/api/flow.proto && \
	protoc -I=$(GODIR)/src -I=$(DIR)/vendor --gogo_out=:. --vine_out=:. $(PACKAGE)/api/rpc.proto

build:
	mkdir -p _output

release: build

lint:
	golint .

clean:
	rm -fr vendor

.PHONY: generate build release clean