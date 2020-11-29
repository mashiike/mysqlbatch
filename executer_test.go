package mysqlbatch_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mashiike/mysqlbatch"
	"github.com/sergi/go-diff/diffmatchpatch"
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
