GIT_VER := $(shell git describe --tags)
DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GO_VER := $(shell go version)

export GO111MODULE := on

.PHONY: test binary install clean dist
cmd/mysqlbatch/mysqlbatch: *.go cmd/mysqlbatch/*.go
	cd cmd/mysqlbatch && go build -ldflags "-s -w -X 'main.Version=${GIT_VER}' -X 'main.BuildDate=${DATE}' -X 'main.GoVersion=${GO_VER}'" -gcflags="-trimpath=${PWD}"

test:
	go test -race .
	go test -race ./cmd/mysqlbatch

clean:
	rm -f cmd/mysqlbatch/mysqlbatch
	rm -fr dist/

dist:
	CGO_ENABLED=0 \
		goxz -pv=$(GIT_VER) \
		-build-ldflags="-s -w -X 'main.Version=${GIT_VER}' -X 'main.BuildDate=${DATE}' -X 'main.GoVersion=${GO_VER}'" \
		-os=darwin,linux -arch=amd64 -d=dist ./cmd/mysqlbatch

release:
	ghr -u mashiike -r mysqlbatch -n "$(GIT_VER)" $(GIT_VER) dist/
