SHELL := /bin/bash
GO_FILES?=$(find . -name '*.go' |grep -v vendor)
PKG_NAME=solidserver

default: build

build:
	go get -v ./...
	go mod tidy
	go mod vendor
	if ! [ -d './_test' ]; then mkdir './_test'; fi
	go build -o ./_test/terraform-provider-solidserver

test: fmtcheck vet
	go test -v ./... || exit 1

fmt:
	gofmt -s -w ./*.go
	gofmt -s -w ./solidserver/*.go

vet:
	go vet -all ./solidserver

fmtcheck:
	./scripts/gofmtcheck.sh
