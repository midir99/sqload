package sqload

import (
	"fmt"
	"strings"
	"testing"
)

func TestExtractSql(t *testing.T) {
	testCases := []struct {
		lines     []string
		wantedSql string
	}{
		{
			[]string{
				"-- Find a user with the given username field",
				"SELECT *",
				"FROM user",
				"WHERE username = 'neto';",
			},
			"SELECT *\nFROM user\nWHERE username = 'neto';",
		},
		{
			[]string{
				"SELECT *",
				"FROM user;",
			},
			"SELECT *\nFROM user;",
		},
		{
			[]string{
				"UPDATE user SET first_name = 'Neto' WHERE id = 78;",
			},
			"UPDATE user SET first_name = 'Neto' WHERE id = 78;",
		},
		{
			[]string{
				"\n\n",
				"DELETE FROM user",
				"WHERE id = 78;",
			},
			"\n\n\nDELETE FROM user\nWHERE id = 78;",
		},
		{
			[]string{
				"",
				"\n\n",
			},
			"\n\n\n",
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			sql := extractSql(testCase.lines)
			if sql != testCase.wantedSql {
				t.Errorf("got %s, want %s", sql, testCase.wantedSql)
				return
			}
		})
	}
}

func TestExtractQueries(t *testing.T) {
	type Want struct {
		queries map[string]string
		err     error
	}
	testCases := []struct {
		sql  string
		want Want
	}{
		{
			strings.Join(
				[]string{
					"-- name: GetUserById",
					"SELECT * FROM user WHERE id = 1;",
					"-- name: GetUserByUsername",
					"SELECT * FROM user WHERE username = 'neto';",
				},
				"\n",
			),
			Want{
				map[string]string{
					"GetUserById":       "SELECT * FROM user WHERE id = 1;",
					"GetUserByUsername": "SELECT * FROM user WHERE username = 'neto';",
				},
				nil,
			},
		},
		{
			strings.Join(
				[]string{
					"--name: GetUserById",
					"",
					"--name: OracionCaribe",
				},
				"\n",
			),
			Want{
				map[string]string{},
				nil,
			},
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			queries, err := extractQueries(testCase.sql)
			if err != testCase.want.err {
				t.Errorf("got %v, want %v", err, testCase.want.err)
				return
			}
			queriesLen := len(queries)
			wantedLen := len(testCase.want.queries)
			if queriesLen != wantedLen {
				t.Errorf("got %v, want %v", testCase.want.queries, queries)
				return
			}
			for queryName, querySql := range queries {
				wantedSql, ok := testCase.want.queries[queryName]
				if !ok {
					t.Errorf("wanted map does not contain key %s", queryName)
					return
				}
				if querySql != wantedSql {
					t.Errorf("got %s, want %s", querySql, wantedSql)
					return
				}
			}
		})
	}
}

func TestFindFilesWithExtension(t *testing.T) {

}

func TestLoadQueriesIntoStruct(t *testing.T) {

}

func TestFromString(t *testing.T) {

}

func TestFromFile(t *testing.T) {

}

func TestFromDir(t *testing.T) {

}
