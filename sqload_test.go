package sqload

import (
	"fmt"
	"os"
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
		{
			"",
			Want{
				map[string]string{},
				nil,
			},
		},
		{
			"-- name: not-a-valid-query-name",
			Want{
				map[string]string{},
				fmt.Errorf("invalid query name: not-a-valid-query-name"),
			},
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			queries, err := extractQueries(testCase.sql)
			if err != nil && fmt.Sprint(err) != fmt.Sprint(testCase.want.err) {
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
	type Want struct {
		files []string
		err   error
	}
	testCases := []struct {
		dir  string
		ext  string
		want Want
	}{
		{
			"testdata/test-find-files-with-extension/",
			".sql",
			Want{
				[]string{
					"testdata/test-find-files-with-extension/dogs.sql",
					"testdata/test-find-files-with-extension/love/u.sql",
					"testdata/test-find-files-with-extension/more-files/even-more-files/random-queries.sql",
				},
				nil,
			},
		},
		{
			"testdata/test-find-files-with-extension/",
			".txt",
			Want{
				[]string{
					"testdata/test-find-files-with-extension/more-files/words-dont-come-easy.txt",
				},
				nil,
			},
		},
		{
			"testdata/test-find-files-with-extension/",
			".txt",
			Want{
				[]string{
					"testdata/test-find-files-with-extension/more-files/words-dont-come-easy.txt",
				},
				nil,
			},
		},
		{
			"testdata/test-find-files-with-extension/",
			".py",
			Want{[]string{}, nil},
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			sqlFiles, err := findFilesWithExtension(testCase.dir, testCase.ext)
			if err != nil && fmt.Sprint(err) != fmt.Sprint(testCase.want.err) {
				t.Errorf("got %v, want %v", err, testCase.want.err)
				return
			}
			sqlFilesLen := len(sqlFiles)
			wantedLen := len(testCase.want.files)
			if sqlFilesLen != wantedLen {
				t.Errorf("got %d, want %d", sqlFilesLen, wantedLen)
				return
			}
			for i := 0; i < sqlFilesLen; i++ {
				if sqlFiles[i] != testCase.want.files[i] {
					t.Errorf("got %v, want %v", sqlFiles, testCase.want.files)
					return
				}
			}
		})
	}
}

func TestLoadQueriesIntoStruct(t *testing.T) {
	// load queries into map
	data, err := os.ReadFile("testdata/cat-queries.sql")
	if err != nil {
		t.Error(err)
	}
	sql := string(data)
	queries, err := extractQueries(sql)
	if err != nil {
		t.Error(err)
	}
	// create test cases to test that the function only accepts pointers to structs
	testCases := []struct {
		v   any
		err error
	}{
		{
			1,
			fmt.Errorf("v is not a pointer"),
		},
		{
			"",
			fmt.Errorf("v is not a pointer"),
		},
		{
			struct{ CreateCatTable string }{},
			fmt.Errorf("v is not a pointer"),
		},
		{
			nil,
			fmt.Errorf("v is nil"),
		},
		{
			map[string]string{},
			fmt.Errorf("v is not a pointer to a struct"),
		},
	}

	// create struct to hold the queries
	type CatQueries struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
		CreateNormalCat string `query:"CreateNormalCat"`
		UpdateColorById string `query:"UpdateColorById"`
	}

	// test using a no-struct type
	err = loadQueriesIntoStruct(queries, 1)
	errMsg := fmt.Sprint(err)
	want := "v is not a pointer"
	if errMsg != want {
		t.Errorf("got %s, want %s", errMsg, want)
	}
	// test using a struct type

}

func TestFromString(t *testing.T) {

}

func TestFromFile(t *testing.T) {

}

func TestFromDir(t *testing.T) {

}
