package mysqlbatch

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/olekukonko/tablewriter"
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
			"%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.User,
			c.Password,
			c.Host,
			c.Port,
			c.Database,
		)
	}
	return strings.TrimPrefix(c.DSN, "mysql://")
}

type Executer struct {
	dsn             string
	lastExecuteTime time.Time
	selectHook      func(query string, columns []string, rows [][]string)
}

func New(config *Config) *Executer {
	return &Executer{
		dsn: config.GetDSN(),
	}
}

func (e *Executer) Execute(queryReader io.Reader) error {
	return e.ExecuteContext(context.Background(), queryReader)
}

func (e *Executer) ExecuteContext(ctx context.Context, queryReader io.Reader) error {

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
		if strings.HasPrefix(strings.ToUpper(query), "SELECT") && e.selectHook != nil {
			if err := e.executeSelect(ctx, db, query); err != nil {
				return errors.Wrap(err, "query rows failed")
			}
			continue
		}
		if _, err := db.ExecContext(ctx, query); err != nil {
			return errors.Wrap(err, "execute query failed")
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.Wrap(scanner.Err(), "query scanner err")
	}
	row := db.QueryRowContext(ctx, "SELECT NOW()")
	if err := row.Err(); err != nil {
		return errors.Wrap(err, "get db time")
	}
	return errors.Wrap(row.Scan(&e.lastExecuteTime), "scan db time")
}

func (e *Executer) executeSelect(ctx context.Context, db *sql.DB, query string) error {
	iter, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer iter.Close()
	columns, err := iter.Columns()
	if err != nil {
		return err
	}
	rows := make([][]string, 0)
	iRow := make([]interface{}, len(columns))
	for iter.Next() {
		row := make([]string, len(columns))
		for i := range row {
			iRow[i] = &row[i]
		}
		if err := iter.Scan(iRow...); err != nil {
			return err
		}
		rows = append(rows, row)
	}
	e.selectHook(query, columns, rows)
	return nil
}

func (e *Executer) LastExecuteTime() time.Time {
	return e.lastExecuteTime
}

func (e *Executer) SetSelectHook(hook func(query string, columns []string, rows [][]string)) {
	e.selectHook = hook
}

func (e *Executer) SetTableSelectHook(hook func(query, table string)) {
	e.selectHook = func(query string, columns []string, rows [][]string) {
		var buf strings.Builder
		tw := tablewriter.NewWriter(&buf)
		tw.SetHeader(columns)
		tw.AppendBulk(rows)
		tw.Render()
		hook(query, buf.String())
	}
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
