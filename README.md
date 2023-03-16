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

## Usage with AWS Lambda (serverless)

Let's solidify the Lambda package with the following zip arcive (runtime `provided.al2`)

```
lambda.zip
├── task.sql
└── bootstrap  
```

A related document is [https://docs.aws.amazon.com/lambda/latest/dg/runtimes-custom.html](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-custom.html)

for example.

deploy lambda functions, in [lambda directory](lambda/)  
The example of lambda directory uses [lambroll](https://github.com/fujiwara/lambroll) for deployment.

For more information on the infrastructure around lambda functions, please refer to [example.tf](lambda/example.tf).

```shell
$ cd lambda/
$ make terraform/init
$ make terraform/plan
$ make terraform/apply
$ make deploy
```

### lambda Payload

for example
```json
{
  "file": "./task.sql",
}
```

output 
```json
{
  "query_results": [
    {
      "Rows": [
        [
          "3",
          "b64ab83358188d4de34fefaa5cf701da@example.com",
          "1"
        ],
        [
          "4",
          "9266d853a5da847cc3355f4b0cd78156@example.com",
          "0"
        ],
        [
          "6",
          "7bc461bfac71283be7cc2612902ec638@example.com",
          "0"
        ],
        [
          "7",
          "a576af3e065e787e691eea537b0eec7b@example.com",
          "0"
        ],
        [
          "8",
          "9c4fd932850bf2026d42ea8844209e6b@example.com",
          "0"
        ]
      ],
      "Columns": [
        "id",
        "name",
        "age"
      ],
      "Query": "SELECT * FROM users WHERE age is NOT NULL LIMIT 5"
    }
  ],
  "last_execute_time": "2023-03-16T10:09:38Z",
  "last_execute_unix_milli": 1678961378000
}
```

## License

see [LICENSE](https://github.com/mashiike/mysqlbatch/blob/master/LICENSE) file.

