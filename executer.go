package mysqlbatch

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"sync"
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
	mu              sync.Mutex
	db              *sql.DB
	lastExecuteTime time.Time
	selectHook      func(query string, columns []string, rows [][]string)
	executeHook     func(query string, rowsAffected int64, lastInsertId int64)
}

func New(config *Config) (*Executer, error) {
	return Open(config.GetDSN())
}

func Open(dsn string) (*Executer, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "mysql connect failed")
	}
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	return &Executer{
		db: db,
	}, nil
}

func (e *Executer) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.db == nil {
		return nil
	}
	return e.db.Close()
}

func (e *Executer) Execute(queryReader io.Reader) error {
	return e.ExecuteContext(context.Background(), queryReader)
}

func (e *Executer) ExecuteContext(ctx context.Context, queryReader io.Reader) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.executeContext(ctx, queryReader); err != nil {
		return err
	}
	return e.updateLastExecuteTime(ctx)
}

func (e *Executer) updateLastExecuteTime(ctx context.Context) error {
	row := e.db.QueryRowContext(ctx, "SELECT NOW()")
	if err := row.Err(); err != nil {
		return errors.Wrap(err, "get db time")
	}
	return errors.Wrap(row.Scan(&e.lastExecuteTime), "scan db time")
}

func (e *Executer) executeContext(ctx context.Context, queryReader io.Reader) error {
	scanner := NewQueryScanner(queryReader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		query := scanner.Query()
		if query == "" {
			continue
		}
		if e.selectHook != nil {
			upperedQuery := strings.ToUpper(query)
			if strings.HasPrefix(upperedQuery, "SELECT") || strings.HasPrefix(upperedQuery, "SHOW") {
				if err := e.queryContext(ctx, query); err != nil {
					return errors.Wrap(err, "query rows failed")
				}
				continue
			}
		}
		result, err := e.db.ExecContext(ctx, query)
		if err != nil {
			return errors.Wrap(err, "execute query failed")
		}
		if e.executeHook != nil {
			lastInsertId, err := result.LastInsertId()
			if err != nil {
				return err
			}
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			e.executeHook(query, rowsAffected, lastInsertId)
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.Wrap(scanner.Err(), "query scanner err")
	}
	return nil
}

func (e *Executer) queryContext(ctx context.Context, query string) error {
	iter, err := e.db.QueryContext(ctx, query)
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
	sRow := make([]sql.NullString, len(columns))
	for i := range sRow {
		iRow[i] = &sRow[i]
	}
	for iter.Next() {
		if err := iter.Scan(iRow...); err != nil {
			return err
		}
		row := make([]string, len(columns))
		for i := range row {
			row[i] = sRow[i].String
		}
		rows = append(rows, row)
	}
	e.selectHook(query, columns, rows)
	return nil
}

func (e *Executer) LastExecuteTime() time.Time {
	return e.lastExecuteTime
}

func (e *Executer) SetExecuteHook(hook func(query string, rowsAffected, lastInsertId int64)) {
	e.executeHook = hook
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
