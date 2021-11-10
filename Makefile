export GO111MODULE := on

.PHONY: test clean
cmd/mysqlbatch/mysqlbatch: *.go cmd/mysqlbatch/*.go
	cd cmd/mysqlbatch && go build .

test:
	go test -race .
	go test -race ./cmd/mysqlbatch

clean:
	rm -f cmd/mysqlbatch/mysqlbatch
	rm -fr dist/
