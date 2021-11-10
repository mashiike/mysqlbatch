package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/mashiike/mysqlbatch"
)

var (
	Version   = "current"
	BuildDate = "(no data)"
)

func main() {
	conf := mysqlbatch.NewDefaultConfig()
	var (
		versionFlag = flag.Bool("v", false, "show version info")
		silentFlag  = flag.Bool("s", false, "no output to console")
		detailFlag  = flag.Bool("d", false, "output deteil for execute sql, -s has priority")
	)
	flag.StringVar(&conf.DSN, "dsn", "", "dsn format as [mysql://]user:pass@tcp(host:port)/dbname (default \"\")")
	flag.StringVar(&conf.User, "u", "root", "username (default root)")
	flag.IntVar(&conf.Port, "P", 3306, "mysql port (default 3306)")
	flag.StringVar(&conf.Password, "p", "", "password")
	flag.StringVar(&conf.Host, "h", "127.0.0.1", "host (default 127.0.0.1)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("version   : %s\n", Version)
		fmt.Printf("go version: %s\n", runtime.Version())
		fmt.Printf("build date: %s\n", BuildDate)
		return
	}

	if flag.NArg() == 1 {
		conf.Database = flag.Arg(0)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM|syscall.SIGHUP|syscall.SIGINT)
	defer stop()

	executer, err := mysqlbatch.New(conf)
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
