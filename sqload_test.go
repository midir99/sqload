package sqload

import (
	"fmt"
	"strings"
	"testing"
)

var CatTestQueries map[string]string = map[string]string{
	"CreateCatTable": strings.TrimSpace(`
CREATE TABLE Cat (
    id SERIAL,
    name VARCHAR(150),
    color VARCHAR(50),

    PRIMARY KEY (id)
);`),
	"CreatePsychoCat": "INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');",
	"CreateNormalCat": "INSERT INTO Cat (name, color) VALUES (:name, :color);",
	"UpdateColorById": strings.TrimSpace(`
UPDATE Cat
   SET color = :color
 WHERE id = :id;
`),
}

var UserTestQueries map[string]string = map[string]string{
	"FindUserById": strings.TrimSpace(`
SELECT first_name,
       last_name,
       dob,
       email
  FROM user
 WHERE id = 1;
`),
	"UpdateFirstNameById": strings.TrimSpace(`
UPDATE user
   SET first_name = 'Ernesto'
 WHERE id = 200;
`),
	"DeleteUserById": strings.TrimSpace(`
DELETE FROM user
      WHERE id = $1;
`),
}

var RiderTestQueries map[string]string = map[string]string{
	"FindRiders": strings.TrimSpace(`
SELECT r.last_name,
       (SELECT MAX(YEAR(championship_date))
          FROM champions AS c
         WHERE c.last_name = r.last_name
           AND c.confirmed = 'Y') AS last_championship_year
  FROM riders AS r
 WHERE r.last_name IN
       (SELECT c.last_name
          FROM champions AS c
         WHERE YEAR(championship_date) > '2008'
           AND c.confirmed = 'Y');
`),
}

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
					"-- query: GetUserById",
					"SELECT * FROM user WHERE id = 1;",
					"-- query: GetUserByUsername",
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
					"--query: GetUserById",
					"",
					"--query: OracionCaribe",
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
			"-- query: not-a-valid-query-name",
			Want{
				map[string]string{},
				fmt.Errorf("invalid query name: not-a-valid-query-name"),
			},
		},
		{
			strings.Join(
				[]string{
					"-- query: ",
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
					" -- query:",
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
				t.Fatalf("got %v, want %v", err, testCase.want.err)
			}
			queriesLen := len(queries)
			wantedLen := len(testCase.want.queries)
			if queriesLen != wantedLen {
				t.Fatalf("got %v, want %v", testCase.want.queries, queries)
			}
			for queryName, querySql := range queries {
				wantedSql, ok := testCase.want.queries[queryName]
				if !ok {
					t.Fatalf("wanted map does not contain key %s", queryName)
				}
				if querySql != wantedSql {
					t.Fatalf("got %s, want %s", querySql, wantedSql)
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
				t.Fatalf("got %v, want %v", err, testCase.want.err)
			}
			sqlFilesLen := len(sqlFiles)
			wantedLen := len(testCase.want.files)
			if sqlFilesLen != wantedLen {
				t.Fatalf("got %d, want %d", sqlFilesLen, wantedLen)
			}
			for i := 0; i < sqlFilesLen; i++ {
				if sqlFiles[i] != testCase.want.files[i] {
					t.Fatalf("got %v, want %v", sqlFiles, testCase.want.files)
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
			fmt.Errorf("v is not a pointer to a struct"),
		},
		{
			"",
			fmt.Errorf("v is not a pointer to a struct"),
		},
		{
			struct{ CreateCatTable string }{},
			fmt.Errorf("v is not a pointer to a struct"),
		},
		{
			nil,
			fmt.Errorf("v is not a pointer to a struct"),
		},
		{
			map[string]string{},
			fmt.Errorf("v is not a pointer to a struct"),
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
	// create a struct that does not use strings
	type InvalidCatQueries struct {
		CreateCatTable int `query:"CreateCatTable"`
	}
	invalidCatQueries := InvalidCatQueries{}
	err := loadQueriesIntoStruct(CatTestQueries, &invalidCatQueries)
	wantedErr := fmt.Errorf("field %s cannot be changed or is not a string", "CreateCatTable")
	if fmt.Sprint(err) != fmt.Sprint(wantedErr) {
		t.Errorf("got %s, want %s", err, wantedErr)
	}
	// create a struct that has a query that the cat-queries.sql file do not
	type MissingCatQueries struct {
		DeleteCatById int `query:"DeleteCatById"`
	}
	missingCatQueries := MissingCatQueries{}
	err = loadQueriesIntoStruct(CatTestQueries, &missingCatQueries)
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
	err = loadQueriesIntoStruct(CatTestQueries, &catQueries)
	if err != nil {
		t.Fatalf("err must be nil, got: %s", err)
	}
	if catQueries.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQueries.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if catQueries.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQueries.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
	}
	if catQueries.CreateNormalCat != CatTestQueries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", catQueries.CreateNormalCat, CatTestQueries["CreateNormalCat"])
	}
	if catQueries.UpdateColorById != CatTestQueries["UpdateColorById"] {
		t.Errorf("got %s, want %s", catQueries.UpdateColorById, CatTestQueries["UpdateColorById"])
	}
}

func TestFromString(t *testing.T) {
	sql := `
	-- query: invalid-name
	`
	err := FromString(sql, 1)
	want := fmt.Errorf("invalid query name: invalid-name")
	if fmt.Sprint(err) != fmt.Sprint(want) {
		t.Errorf("got %s, want %s", err, want)
	}
	sql = strings.TrimSpace(`
-- query: CreateCatTable
CREATE TABLE Cat (
    id SERIAL,
    name VARCHAR(150),
    color VARCHAR(50),

    PRIMARY KEY (id)
);
-- query: CreatePsychoCat
INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');`)
	type CatQueries struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
	}
	catQueries := CatQueries{}
	err = FromString(sql, &catQueries)
	if err != nil {
		t.Fatalf("err must be nil, got: %s", err)
	}
	if catQueries.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQueries.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if catQueries.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQueries.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
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
		t.Fatalf("error loading testdata/cat-queries.sql: %s", err)
	}
	if catQueries.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQueries.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if catQueries.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQueries.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
	}
	if catQueries.CreateNormalCat != CatTestQueries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", catQueries.CreateNormalCat, CatTestQueries["CreateNormalCat"])
	}
	if catQueries.UpdateColorById != CatTestQueries["UpdateColorById"] {
		t.Errorf("got %s, want %s", catQueries.UpdateColorById, CatTestQueries["UpdateColorById"])
	}
}

func TestFromDir(t *testing.T) {
	type RandomQueries struct {
		CreateCatTable      string `query:"CreateCatTable"`
		CreatePsychoCat     string `query:"CreatePsychoCat"`
		CreateNormalCat     string `query:"CreateNormalCat"`
		UpdateColorById     string `query:"UpdateColorById"`
		FindUserById        string `query:"FindUserById"`
		UpdateFirstNameById string `query:"UpdateFirstNameById"`
		DeleteUserById      string `query:"DeleteUserById"`
		FindRiders          string `query:"FindRiders"`
	}
	queries := RandomQueries{}
	err := FromDir("testdata/i-dont-exist/", &queries)
	if err == nil {
		t.Errorf("dir testdata/i-dont-exist/ must not exists so this test can fail")
	}
	err = FromDir("testdata/test-from-dir/", &queries)
	if err != nil {
		t.Fatalf("error loading testdata/test-from-dir/: %s", err)
	}
	if queries.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", queries.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if queries.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", queries.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
	}
	if queries.CreateNormalCat != CatTestQueries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", queries.CreateNormalCat, CatTestQueries["CreateNormalCat"])
	}
	if queries.UpdateColorById != CatTestQueries["UpdateColorById"] {
		t.Errorf("got %s, want %s", queries.UpdateColorById, CatTestQueries["UpdateColorById"])
	}
	if queries.FindUserById != UserTestQueries["FindUserById"] {
		t.Errorf("got %s, want %s", queries.FindUserById, UserTestQueries["FindUserById"])
	}
	if queries.UpdateFirstNameById != UserTestQueries["UpdateFirstNameById"] {
		t.Errorf("got %s, want %s", queries.UpdateFirstNameById, UserTestQueries["UpdateFirstNameById"])
	}
	if queries.DeleteUserById != UserTestQueries["DeleteUserById"] {
		t.Errorf("got %s, want %s", queries.DeleteUserById, UserTestQueries["DeleteUserById"])
	}
	if queries.FindRiders != RiderTestQueries["FindRiders"] {
		t.Errorf("got %s, want %s", queries.FindRiders, RiderTestQueries["FindRiders"])
	}
}
