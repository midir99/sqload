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
		{
			strings.Join(
				[]string{
					"-- name: ",
				},
				"\n",
			),
			Want{
				map[string]string{},
				fmt.Errorf("invalid query name: "),
			},
		},
		{
			strings.Join(
				[]string{
					" -- name:",
					"EmptyQuery",
				},
				"\n",
			),
			Want{
				map[string]string{
					"EmptyQuery": "",
				},
				nil,
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
	// create test cases to test that the function only accepts pointers to structs
	var nilPtr *int = nil
	num := 1
	intPtr := &num
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
			fmt.Errorf("v is not a pointer"),
		},
		{
			map[string]string{},
			fmt.Errorf("v is not a pointer"),
		},
		{
			nilPtr,
			fmt.Errorf("v is nil"),
		},
		{
			intPtr,
			fmt.Errorf("v is not a pointer to a struct"),
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d (v=%v)", i, testCase.v), func(t *testing.T) {
			err := loadQueriesIntoStruct(map[string]string{}, testCase.v)
			if fmt.Sprint(err) != fmt.Sprint(testCase.err) {
				t.Errorf("got %s, want %s", err, testCase.err)
				return
			}
		})
	}
	// load queries into map
	data, err := os.ReadFile("testdata/cat-queries.sql")
	if err != nil {
		t.Error(err)
	}
	sql := string(data)
	queries, err := extractQueries(sql)
	if err != nil {
		t.Error("test cannot continue if loading queries fails")
	}
	// create a struct that does not use strings
	type InvalidCatQueries struct {
		CreateCatTable int `query:"CreateCatTable"`
	}
	invalidCatQueries := InvalidCatQueries{}
	err = loadQueriesIntoStruct(queries, &invalidCatQueries)
	wantedErr := fmt.Errorf("field %s cannot be changed or is not a string", "CreateCatTable")
	if fmt.Sprint(err) != fmt.Sprint(wantedErr) {
		t.Errorf("got %s, want %s", err, wantedErr)
	}
	// create a struct that has a query that the cat-queries.sql file do not
	type MissingCatQueries struct {
		DeleteCatById int `query:"DeleteCatById"`
	}
	missingCatQueries := MissingCatQueries{}
	err = loadQueriesIntoStruct(queries, &missingCatQueries)
	wantedErr = fmt.Errorf("could not to find query %s", "DeleteCatById")
	if fmt.Sprint(err) != fmt.Sprint(wantedErr) {
		t.Errorf("got %s, want %s", err, wantedErr)
	}
	// create struct to hold the queries
	type CatQueries struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
		CreateNormalCat string `query:"CreateNormalCat"`
		UpdateColorById string `query:"UpdateColorById"`
	}
	catQueries := CatQueries{}
	loadQueriesIntoStruct(queries, &catQueries)
	if catQueries.CreateCatTable != queries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQueries.CreateCatTable, queries["CreateCatTable"])
	}
	if catQueries.CreatePsychoCat != queries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQueries.CreatePsychoCat, queries["CreatePsychoCat"])
	}
	if catQueries.CreateNormalCat != queries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", catQueries.CreateNormalCat, queries["CreateNormalCat"])
	}
	if catQueries.UpdateColorById != queries["UpdateColorById"] {
		t.Errorf("got %s, want %s", catQueries.UpdateColorById, queries["UpdateColorById"])
	}
}

func TestFromString(t *testing.T) {
	sql := `
	-- name: invalid-name
	`
	err := FromString(sql, 1)
	want := fmt.Errorf("invalid query name: invalid-name")
	if fmt.Sprint(err) != fmt.Sprint(want) {
		t.Errorf("got %s, want %s", err, want)
	}
	sql = `
	-- name: CreateCatTable
	CREATE TABLE Cat (
		id SERIAL,
		name VARCHAR(150),
		color VARCHAR(50),

		PRIMARY KEY (id)
	)
	-- name: CreatePsychoCat
	INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');
	`
	type CatQueries struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
	}
	catQueries := CatQueries{}
	err = FromString(sql, &catQueries)
	if err != nil {
		t.Error("err must be nil")
	}
	queries, err := extractQueries(sql)
	if err != nil {
		t.Error("test cannot continue if loading queries fails")
	}
	if catQueries.CreateCatTable != queries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQueries.CreateCatTable, queries["CreateCatTable"])
	}
	if catQueries.CreatePsychoCat != queries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQueries.CreatePsychoCat, queries["CreatePsychoCat"])
	}
}

func TestFromFile(t *testing.T) {
	type CatQueries struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
		CreateNormalCat string `query:"CreateNormalCat"`
		UpdateColorById string `query:"UpdateColorById"`
	}
	catQueries := CatQueries{}
	err := FromFile("testdata/i-dont-exist.sql", &catQueries)
	if err == nil {
		t.Errorf("file testdata/i-dont-exist.sql must not exists so this test can fail")
	}
	err = FromFile("testdata/cat-queries.sql", &catQueries)
	if err != nil {
		t.Errorf("error loading testdata/cat-queries.sql: %s", err)
	}
	data, err := os.ReadFile("testdata/cat-queries.sql")
	if err != nil {
		t.Errorf("test cannot continue if reading file fails: %s", err)
	}
	queries, err := extractQueries(string(data))
	if err != nil {
		t.Error("test cannot continue if loading queries fails")
	}
	if catQueries.CreateCatTable != queries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQueries.CreateCatTable, queries["CreateCatTable"])
	}
	if catQueries.CreatePsychoCat != queries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQueries.CreatePsychoCat, queries["CreatePsychoCat"])
	}
	if catQueries.CreateNormalCat != queries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", catQueries.CreateNormalCat, queries["CreateNormalCat"])
	}
	if catQueries.UpdateColorById != queries["UpdateColorById"] {
		t.Errorf("got %s, want %s", catQueries.UpdateColorById, queries["UpdateColorById"])
	}
}

func TestFromDir(t *testing.T) {

}
