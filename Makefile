GIT_VER := $(shell git describe --tags)
DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GO_VER := $(shell go version)

export GO111MODULE := on

.PHONY: test binary install clean dist
cmd/mysqlbatch/mysqlbatch: *.go cmd/mysqlbatch/*.go
	cd cmd/mysqlbatch && go build .

test:
	go test -race .
	go test -race ./cmd/mysqlbatch

clean:
	rm -f cmd/mysqlbatch/mysqlbatch
	rm -fr dist/
