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

// Config is a connection setting to MySQL.
// Exists to generate a Golang connection DSN to MySQL
type Config struct {
	DSN      string
	User     string
	Password string
	Host     string
	Port     int
	Database string
}

// NewDefaultConfig returns the config for connecting to the local MySQL server
func NewDefaultConfig() *Config {
	return &Config{
		User: "root",
		Host: "127.0.0.1",
		Port: 3306,
	}
}

//GetDSN returns a DSN dedicated to connecting to MySQL.
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

//Executer queries the DB. There is no parallelism
type Executer struct {
	mu              sync.Mutex
	db              *sql.DB
	lastExecuteTime time.Time
	selectHook      func(query string, columns []string, rows [][]string)
	executeHook     func(query string, rowsAffected int64, lastInsertId int64)
	isSelectFunc    func(query string) bool
	timeCheckQuery  string
}

//New return Executer with config
func New(config *Config) (*Executer, error) {
	return Open(config.GetDSN())
}

//Open with dsn
func Open(dsn string) (*Executer, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "mysql connect failed")
	}
	return NewWithDB(db), nil
}

// NewWithDB returns Executer with *sql.DB
// Note: Since it is made assuming MySQL, it may be inconvenient for other DBs.
func NewWithDB(db *sql.DB) *Executer {
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	return &Executer{
		db:             db,
		timeCheckQuery: "SELECT NOW()",
	}
}

// Close DB
func (e *Executer) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.db == nil {
		return nil
	}
	return e.db.Close()
}

//Execute SQL
func (e *Executer) Execute(queryReader io.Reader) error {
	return e.ExecuteContext(context.Background(), queryReader)
}

//ExecuteContext SQL execute with context.Context
func (e *Executer) ExecuteContext(ctx context.Context, queryReader io.Reader) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.executeContext(ctx, queryReader); err != nil {
		return err
	}
	return e.updateLastExecuteTime(ctx)
}

func (e *Executer) updateLastExecuteTime(ctx context.Context) error {
	row := e.db.QueryRowContext(ctx, e.timeCheckQuery)
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
			var isSelect bool
			if e.isSelectFunc == nil {
				if strings.HasPrefix(upperedQuery, "SELECT") || strings.HasPrefix(upperedQuery, "SHOW") || strings.HasPrefix(upperedQuery, `\`) {
					isSelect = true
				}
			} else {
				isSelect = e.isSelectFunc(upperedQuery)
			}
			if isSelect {
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

//LastExecuteTime returns last execute time on DB
func (e *Executer) LastExecuteTime() time.Time {
	return e.lastExecuteTime
}

//SetExecuteHook set non select query hook
func (e *Executer) SetExecuteHook(hook func(query string, rowsAffected, lastInsertId int64)) {
	e.executeHook = hook
}

//SetSelectHook set select query hook
func (e *Executer) SetSelectHook(hook func(query string, columns []string, rows [][]string)) {
	e.selectHook = hook
}

//SetIsSelectFunc :Set the function to decide whether to execute in QueryContext
func (e *Executer) SetIsSelectFunc(f func(query string) bool) {
	e.isSelectFunc = f
}

//SetTimeCheckQuery set time check query for non mysql db
func (e *Executer) SetTimeCheckQuery(query string) {
	e.timeCheckQuery = query
}

//SetTimeCheckQuery set select query hook, but result is table string
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

// QueryScanner separate string by ; and delete newline
type QueryScanner struct {
	*bufio.Scanner
}

//NewQueryScanner returns QueryScanner
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

//Query return
func (s *QueryScanner) Query() string {
	return strings.Trim(strings.NewReplacer(
		"\r\n", " ",
		"\r", " ",
		"\n", " ",
	).Replace(s.Text()), " \t")
}
