package mysqlbatch_test

import (
	"strings"
	"testing"

	"github.com/mashiike/mysqlbatch"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestQueryScanner(t *testing.T) {

	input := `
UPDATE user_credentials
SET password_hash = "";
UPDATE user_profiles SET email = concat(MD5(email), '@localhost');
DELETE FROM roles

`
	queries := []string{
		`UPDATE user_credentials SET password_hash = ""`,
		`UPDATE user_profiles SET email = concat(MD5(email), '@localhost')`,
		`DELETE FROM roles`,
	}

	scanner := mysqlbatch.NewQueryScanner(strings.NewReader(input))

	i := 0
	for ; scanner.Scan(); i++ {
		query := scanner.Query()
		if i >= len(queries) {
			t.Fatalf("unexpected over query count: %d", i)
		}
		if diff, same := diffStr(query, queries[i]); !same {
			t.Logf("got     : %s", query)
			t.Logf("expected: %s", queries[i])
			t.Errorf("unexpected query diff: %s", diff)
		}
	}
	if i != len(queries) {
		t.Errorf("unexpected count: %d", i)
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
