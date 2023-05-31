package mysqlbatch_test

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mashiike/mysqlbatch"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/require"
)

func TestQueryScanner(t *testing.T) {

	cases := []struct {
		input   string
		queries []string
	}{
		{
			input: `
UPDATE user_credentials
SET password_hash = "";
UPDATE user_profiles SET email = concat(MD5(email), '@localhost');
DELETE FROM roles
`,
			queries: []string{
				`UPDATE user_credentials SET password_hash = ""`,
				`UPDATE user_profiles SET email = concat(MD5(email), '@localhost')`,
				`DELETE FROM roles`,
			},
		},
		{
			input: `
BEGIN;

SET FOREIGN_KEY_CHECKS = 0;

SET FOREIGN_KEY_CHECKS = 1;

COMMIT;
`,
			queries: []string{
				`BEGIN`,
				`SET FOREIGN_KEY_CHECKS = 0`,
				`SET FOREIGN_KEY_CHECKS = 1`,
				`COMMIT`,
				``,
			},
		},
	}

	for casenum, c := range cases {
		t.Run(fmt.Sprintf("case.%d", casenum), func(t *testing.T) {
			scanner := mysqlbatch.NewQueryScanner(strings.NewReader(c.input))

			i := 0
			for ; scanner.Scan(); i++ {
				query := scanner.Query()
				if i >= len(c.queries) {
					t.Logf("query: %s", query)
					t.Fatalf("unexpected over query count: %d", i)
				}
				if diff, same := diffStr(query, c.queries[i]); !same {
					t.Logf("got     : %s", query)
					t.Logf("expected: %s", c.queries[i])
					t.Errorf("unexpected query diff: %s", diff)
				}
			}
			if i != len(c.queries) {
				t.Errorf("unexpected count: %d", i)
			}
		})
	}
}

func diffStr(a, b string) (string, bool) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(b, a, true)
	var same bool = true
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			same = false
		}
	}
	return dmp.DiffPrettyText(diffs), same
}

//go:embed testdata/test.sql
var testSQL []byte

//go:embed testdata/test_template.sql
var testTemplateSQL []byte

func TestExecuterExecute(t *testing.T) {
	conf := mysqlbatch.NewDefaultConfig()
	conf.Password = "mysqlbatch"
	conf.Location = "Asia/Tokyo"
	e, err := mysqlbatch.New(context.Background(), conf)
	require.NoError(t, err)
	defer e.Close()

	e.SetTimeCheckQuery("SELECT NOW()")
	e.SetTableSelectHook(func(query, table string) {
		t.Log(query + "\n" + table + "\n")
	})
	var count int32
	e.SetSelectHook(func(query string, columns []string, rows [][]string) {
		atomic.AddInt32(&count, 1)
		require.Equal(t, `SELECT * FROM users WHERE age is NOT NULL LIMIT 5`, query)
		require.Equal(t, 5, len(rows))
	})
	err = e.Execute(bytes.NewReader(testSQL), nil)
	require.NoError(t, err)
	log.Println("LastExecuteTime:", e.LastExecuteTime())
	require.InDelta(t, time.Since(e.LastExecuteTime()), 0, float64(5*time.Minute))
	require.EqualValues(t, 1, count)
}

func TestExecuterExecute__WithVars(t *testing.T) {
	os.Setenv("ENV", "test")
	mysqlbatch.DefaultSQLDumper = os.Stderr
	conf := mysqlbatch.NewDefaultConfig()
	conf.Password = "mysqlbatch"
	conf.Location = "Asia/Tokyo"
	e, err := mysqlbatch.New(context.Background(), conf)
	require.NoError(t, err)
	defer e.Close()

	e.SetTimeCheckQuery("SELECT NOW()")
	e.SetTableSelectHook(func(query, table string) {
		t.Log(query + "\n" + table + "\n")
	})
	var count int32
	e.SetSelectHook(func(query string, columns []string, rows [][]string) {
		atomic.AddInt32(&count, 1)
		require.Equal(t, `SELECT * FROM users WHERE age is NOT NULL LIMIT 5`, query)
		require.Equal(t, 5, len(rows))
	})
	err = e.Execute(bytes.NewReader(testTemplateSQL), map[string]string{
		"relation":      "users",
		"limit":         "5",
		"age_condition": " > 20",
	})
	require.NoError(t, err)
	log.Println("LastExecuteTime:", e.LastExecuteTime())
	require.InDelta(t, time.Since(e.LastExecuteTime()), 0, float64(5*time.Minute))
	require.EqualValues(t, 1, count)
}
