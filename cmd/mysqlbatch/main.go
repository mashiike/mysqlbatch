package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mashiike/mysqlbatch"
)

var (
	Version   = "current"
	BuildDate = "(no data)"
	GoVersion = "(no data)"
)

func main() {
	conf := mysqlbatch.NewDefaultConfig()
	var (
		versionFlag = flag.Bool("v", false, "show version info")
	)
	flag.StringVar(&conf.User, "u", "root", "username (default root)")
	flag.IntVar(&conf.Port, "P", 3306, "mysql port (default 3306)")
	flag.StringVar(&conf.Password, "p", "", "password")
	flag.StringVar(&conf.Host, "h", "127.0.0.1", "host (default 127.0.0.1)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("version   : %s\n", Version)
		fmt.Printf("go version: %s\n", GoVersion)
		fmt.Printf("build date: %s\n", BuildDate)
		return
	}

	if flag.NArg() == 1 {
		conf.Database = flag.Arg(0)
	}

	executer := mysqlbatch.New(conf)
	if err := executer.Execute(os.Stdin); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}