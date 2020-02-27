package main

import (
	"flag"
	"log"
	"os"

	"github.com/mashiike/mysqlbatch"
)

func main() {
	conf := mysqlbatch.NewDefaultConfig()
	flag.StringVar(&conf.User, "u", "root", "username (default root)")
	flag.IntVar(&conf.Port, "P", 3306, "mysql port (default 3306)")
	flag.StringVar(&conf.Password, "p", "", "password")
	flag.StringVar(&conf.Host, "h", "127.0.0.1", "host (default 127.0.0.1)")
	flag.Parse()

	if flag.NArg() == 1 {
		conf.Database = flag.Arg(0)
	}

	executer := mysqlbatch.New(conf)
	if err := executer.Execute(os.Stdin); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
