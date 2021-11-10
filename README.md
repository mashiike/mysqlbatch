![](https://github.com/mashiike/mysqlbatch/workflows/Test/badge.svg)


# mysqlbatch
mysqlbatch accepts multiple queries from standard input.
Just like the standard mysql command batch mode.  

mysqlbatch can be bundled with Docker, [AWS Lambda](https://aws.amazon.com/jp/lambda/) Function, etc. for one binary.


I created it because I wanted to issue a query from AWS Lambda Function on VPC to RDS Aurora (MySQL compatible) using [Bash Layer](https://github.com/gkrizek/bash-lambda-layer).


## Install

### Homebrew (macOS only)

```
$ brew install mashiike/tap/mysqlbatch
```


### Binary packages

[Releases](https://github.com/mashiike/mysqlbatch/releases)


## Simple usecase

like mysql-client for batch mode.

as ...
```
$ mysqlbatch -u root -p ${password} -h localhost < batch.sql
```


## Usage as a library


```go
executer, err := mysqlbatch.Open("root:password@tcp(localhost:3306)/testdb?parseTime=true")
if err != nil {
    //...
}
defer executer.Close()
if err := executer.Execute(strings.NewReader("UPDATE users SET name = 'hoge';")); err != nil {
    //...
}
```

more infomation see [go doc](https://godoc.org/github.com/mashiike/mysqlbatch).

## License

see [LICENSE](https://github.com/mashiike/mysqlbatch/blob/master/LICENSE) file.

