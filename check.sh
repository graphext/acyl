#!/bin/bash -x

set -e

go build
go test -cover $(go list ./... |grep -v pkg/persistence |grep -v pkg/api)
go test -cover github.com/dollarshaveclub/acyl/pkg/persistence
go test -cover github.com/dollarshaveclub/acyl/pkg/api
DOCKER_BUILDKIT=1 docker build -t at .