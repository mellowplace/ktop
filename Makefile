.PHONY: update-deps

update-deps:
	glide update -v

install-deps:
	glide install -v

build:
	go build -o ./bin/ktop ./cmd/ktop/main.go
