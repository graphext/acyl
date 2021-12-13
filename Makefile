.PHONY: build generate test

default: build

build:
	go mod vendor
	go install github.com/dollarshaveclub/acyl

generate:
	go generate ./...

check:
	./check.sh

docs:
	./openapi.sh
