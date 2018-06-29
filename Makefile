GOOS ?= linux
GOARCH ?= amd64
LDFLAGS ?= -ldflags "-linkmode external -extldflags -static"

.PHONY: update-deps

update-deps:
	glide update -v

install-deps:
	glide install -v

build:
	go build ${LDFLAGS} -o ./dist/${GOOS}/${GOARCH}/ktop ./cmd/ktop/main.go

cross-compile:
	$(MAKE) build GOOS=linux GOARCH=amd64
	$(MAKE) build GOOS=darwin GOARCH=amd64 LDFLAGS=
	$(MAKE) build GOOS=windows GOARCH=amd64 LDFLAGS=

install:
	go install ./cmd/...
