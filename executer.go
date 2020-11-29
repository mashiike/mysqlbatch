package mysqlbatch

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

type Config struct {
	DSN      string
	User     string
	Password string
	Host     string
	Port     int
	Database string
}

func NewDefaultConfig() *Config {
	return &Config{
		User: "root",
		Host: "127.0.0.1",
		Port: 3306,
	}
}

func (c *Config) GetDSN() string {
	if c.DSN == "" {
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s",
			c.User,
			c.Password,
			c.Host,
			c.Port,
			c.Database,
		)
	}
	return strings.TrimLeft(c.DSN, "mysql://")
}

type Executer struct {
	dsn string
}

func New(config *Config) *Executer {
	return &Executer{
		dsn: config.GetDSN(),
	}
}

func (e *Executer) Execute(queryReader io.Reader) error {

	db, err := sql.Open("mysql", e.dsn)
	if err != nil {
		return errors.Wrap(err, "mysql connect failed")
	}
	defer db.Close()
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	scanner := NewQueryScanner(queryReader)
	for scanner.Scan() {
		query := scanner.Query()
		if query == "" {
			continue
		}
		if _, err := db.Exec(query); err != nil {
			return errors.Wrap(err, "execute query failed")
		}
	}
	return errors.Wrap(scanner.Err(), "query scanner err")
}

type QueryScanner struct {
	*bufio.Scanner
}

func NewQueryScanner(queryReader io.Reader) *QueryScanner {
	scanner := bufio.NewScanner(queryReader)
	onSplit := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		for i := 0; i < len(data); i++ {
			if data[i] == ';' {
				return i + 1, data[:i], nil
			}
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, data, bufio.ErrFinalToken
	}
	scanner.Split(onSplit)
	return &QueryScanner{
		Scanner: scanner,
	}
}

func (s *QueryScanner) Query() string {
	return strings.Trim(strings.NewReplacer(
		"\r\n", " ",
		"\r", " ",
		"\n", " ",
	).Replace(s.Text()), " \t")
}
