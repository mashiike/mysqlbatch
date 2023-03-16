package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/ken39arg/go-flagx"
	"github.com/mashiike/mysqlbatch"
)

var (
	Version   = "current"
	BuildDate = "(no data)"
)

func main() {
	conf := mysqlbatch.NewDefaultConfig()
	var (
		versionFlag         = flag.Bool("v", false, "show version info")
		silentFlag          = flag.Bool("s", false, "no output to console")
		detailFlag          = flag.Bool("d", false, "output deteil for execute sql, -s has priority")
		enableBootstrapFlag = flag.Bool("enable-lambda-bootstrap", false, "if run on AWS Lambda, running as lambda bootstrap")
	)
	flag.StringVar(&conf.DSN, "dsn", "", "dsn format as [mysql://]user:pass@tcp(host:port)/dbname (default \"\")")
	flag.StringVar(&conf.User, "u", "root", "username (default root)")
	flag.StringVar(&conf.User, "user", "root", "")
	flag.IntVar(&conf.Port, "P", 3306, "mysql port (default 3306)")
	flag.IntVar(&conf.Port, "port", 3306, "")
	flag.StringVar(&conf.Password, "p", "", "password")
	flag.StringVar(&conf.Password, "password", "", "")
	flag.StringVar(&conf.Host, "h", "127.0.0.1", "host (default 127.0.0.1)")
	flag.StringVar(&conf.Host, "host", "", "")
	flag.VisitAll(flagx.EnvToFlagWithPrefix("MYSQLBATCH_"))
	flag.Parse()

	if *versionFlag {
		fmt.Printf("version   : %s\n", Version)
		fmt.Printf("go version: %s\n", runtime.Version())
		fmt.Printf("build date: %s\n", BuildDate)
		return
	}
	conf.Database = os.Getenv("MYSQLBATCH_DATABASE")
	if flag.NArg() == 1 {
		conf.Database = flag.Arg(0)
	}
	if *enableBootstrapFlag && strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") || os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		h := handler{
			conf: conf,
		}
		lambda.StartWithOptions(h.Invoke)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM|syscall.SIGHUP|syscall.SIGINT)
	defer stop()

	executer, err := mysqlbatch.New(ctx, conf)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}
	defer executer.Close()
	if !*silentFlag {
		executer.SetTableSelectHook(func(query, table string) {
			log.Println(query + "\n" + table + "\n")
		})
		if *detailFlag {
			executer.SetExecuteHook(func(query string, rowsAffected, lastInsertId int64) {
				log.Printf("%s\nQuery OK, %d rows affected, last inserted id = %d\n", query, rowsAffected, lastInsertId)
			})
		}
	}
	if err := executer.ExecuteContext(ctx, os.Stdin); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if !*silentFlag {
		log.Println("DB time when the last SQL was executed:", executer.LastExecuteTime())
	}
}

type handler struct {
	conf *mysqlbatch.Config
}

type payload struct {
	SQL      string  `json:"sql,omitempty"`
	File     string  `json:"file,omitempty"`
	DSN      *string `json:"dsn,omitempty"`
	User     *string `json:"user,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Host     *string `json:"host,omitempty"`
	Database *string `json:"database,omitempty"`
}

type response struct {
	QueryResults         []queryResults `json:"query_results,omitempty"`
	LastExecuteTime      time.Time      `json:"last_execute_time,omitempty"`
	LastExecuteUnixMilli int64          `json:"last_execute_unix_milli,omitempty"`
}

type queryResults struct {
	Rows    [][]string
	Columns []string
	Query   string
}

func (h *handler) Invoke(ctx context.Context, p *payload) (*response, error) {
	conf := *h.conf
	if p.DSN != nil {
		conf.DSN = *p.DSN
	}
	if p.User != nil {
		conf.User = *p.User
	}
	if p.Port != nil {
		conf.Port = *p.Port
	}
	if p.Host != nil {
		conf.Host = *p.Host
	}
	if p.Database != nil {
		conf.Database = *p.Database
	}
	executer, err := mysqlbatch.New(ctx, &conf)
	if err != nil {
		return nil, err
	}
	defer executer.Close()
	var query io.Reader
	if p.File != "" {
		fp, err := os.Open(p.File)
		if err != nil {
			return nil, err
		}
		defer fp.Close()
		query = fp
	} else if p.SQL != "" {
		query = strings.NewReader(p.SQL)
	} else {
		log.Println("nothing todo")
		return &response{}, nil
	}
	var mu sync.Mutex
	var results []queryResults
	executer.SetSelectHook(func(query string, columns []string, rows [][]string) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, queryResults{
			Rows:    rows,
			Columns: columns,
			Query:   query,
		})
	})
	if err := executer.ExecuteContext(ctx, query); err != nil {
		return nil, err
	}
	r := &response{
		QueryResults:         results,
		LastExecuteTime:      executer.LastExecuteTime(),
		LastExecuteUnixMilli: executer.LastExecuteTime().UnixMilli(),
	}
	return r, nil
}
