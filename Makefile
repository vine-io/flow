NAME=$(shell echo "flow")
GODIR=$(shell go env GOPATH)
DIR=$(shell pwd)
PACKAGE=github.com/vine-io/flow
GIT_COMMIT=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --abbrev=0 --tags --always --match "v*")
GIT_VERSION=github.com/vine-io/flow/pkg/internal/doc
CGO_ENABLED=0
BUILD_DATE=$(shell date +%s)
TOOLS=$(shell echo "protoc-gen-flow" )

changelog:
	mkdir -p _output
	changelog --last --output _output/CHANGELOG.md

generate:
	cd $(GODIR)/src && \
	protoc -I=$(GODIR)/src -I=$(DIR)/vendor --gogo_out=:. --deepcopy_out=:. --validator_out=:. $(PACKAGE)/api/flow.proto && \
	protoc -I=$(GODIR)/src -I=$(DIR)/vendor --gogo_out=:. --validator_out=:. --vine_out=:. $(PACKAGE)/api/rpc.proto

build:
	mkdir -p _output

tar-windows:
	mkdir -p _output/windows-amd64
	for i in $(TOOLS); do \
	    GOOS=windows GOARCH=amd64 go build -a -installsuffix cgo -ldflags "-s -w ${LDFLAGS}" -o _output/windows-amd64/$$i.exe cmd/$$i/main.go ;\
	done && \
	cd _output && rm -fr $(NAME)-windows-amd64-$(GIT_TAG).zip && zip $(NAME)-windows-amd64-$(GIT_TAG).zip windows-amd64/* && rm -fr windows-amd64

tar-linux-amd64:
	mkdir -p _output/linux-amd64
	for i in $(TOOLS); do \
	    GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags "-s -w ${LDFLAGS}" -o _output/linux-amd64/$$i cmd/$$i/main.go ;\
	done && \
	cd _output && rm -fr $(NAME)-linux-amd64-$(GIT_TAG).tar.gz && tar -zcvf $(NAME)-linux-amd64-$(GIT_TAG).tar.gz linux-amd64/* && rm -fr linux-amd64

tar-linux-arm64:
	mkdir -p _output/linux-arm64
	for i in $(TOOLS); do \
	    GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -ldflags "-s -w ${LDFLAGS}" -o _output/linux-arm64/$$i cmd/$$i/main.go ;\
	done && \
	cd _output && rm -fr $(NAME)-linux-arm64-$(GIT_TAG).tar.gz && tar -zcvf $(NAME)-linux-arm64-$(GIT_TAG).tar.gz linux-arm64/* && rm -fr linux-arm64

tar-darwin-amd64:
	mkdir -p _output/darwin-amd64
	for i in $(TOOLS); do \
	    GOOS=darwin GOARCH=amd64 go build -a -installsuffix cgo -ldflags "-s -w ${LDFLAGS}" -o _output/darwin-amd64/$$i cmd/$$i/main.go ;\
	done && \
	cd _output && rm -fr $(NAME)-darwin-amd64-$(GIT_TAG).tar.gz && tar -zcvf $(NAME)-darwin-amd64-$(GIT_TAG).tar.gz darwin-amd64/* && rm -fr darwin-amd64

tar-darwin-arm64:
	mkdir -p _output/darwin-arm64
	for i in $(TOOLS); do \
	    GOOS=darwin GOARCH=arm64 go build -a -installsuffix cgo -ldflags "-s -w ${LDFLAGS}" -o _output/darwin-arm64/$$i cmd/$$i/main.go ;\
	done && \
	cd _output && rm -fr $(NAME)-darwin-arm64-$(GIT_TAG).tar.gz && tar -zcvf $(NAME)-darwin-arm64-$(GIT_TAG).tar.gz darwin-arm64/* && rm -fr darwin-arm64

tar: changelog tar-windows tar-linux-amd64 tar-linux-arm64 tar-darwin-amd64 tar-darwin-arm64

build:
	go build -a -installsuffix cgo -ldflags "-s -w $(LDFLAGS)" -o $(NAME) cmd/vine/main.go

release: build

lint:
	golint .

clean:
	rm -fr vendor

.PHONY: generate build release clean