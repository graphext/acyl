.PHONY: build generate test

default: build

build:
	GO111MODULE=off go install github.com/dollarshaveclub/acyl

generate:
	GO111MODULE=off go generate ./...

check:
	./check.sh