package sqload

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
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

func TestExtractQueryMap(t *testing.T) {
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
				fmt.Errorf("%w: invalid query name not-a-valid-query-name", ErrCannotLoadQueries),
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
				fmt.Errorf("%w: invalid query name ", ErrCannotLoadQueries),
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
			queries, err := ExtractQueryMap(testCase.sql)
			if err != nil {
				if !errors.Is(err, ErrCannotLoadQueries) {
					t.Fatalf("error %v does not wrap %v", err, ErrCannotLoadQueries)
				}
				if testCase.want.err == nil {
					t.Fatalf("got %v, want no error", err)
				}
				if err.Error() != testCase.want.err.Error() {
					t.Fatalf("got %v, want %v", err, testCase.want.err)
				}
			} else if testCase.want.err != nil {
				t.Fatalf("got no error, want %v", testCase.want.err)
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

func TestFindFilesWithExt(t *testing.T) {
	type Want struct {
		files []string
		err   error
	}
	testCases := []struct {
		fsys fs.FS
		ext  string
		want Want
	}{
		{
			os.DirFS("testdata/test-find-files-with-ext/"),
			".sql",
			Want{
				[]string{
					"dogs.sql",
					"love/u.sql",
					"more-files/even-more-files/random-queries.sql",
				},
				nil,
			},
		},
		{
			os.DirFS("testdata/test-find-files-with-ext/"),
			".txt",
			Want{
				[]string{
					"more-files/words-dont-come-easy.txt",
				},
				nil,
			},
		},
		{
			os.DirFS("testdata/test-find-files-with-ext/"),
			".txt",
			Want{
				[]string{
					"more-files/words-dont-come-easy.txt",
				},
				nil,
			},
		},
		{
			os.DirFS("testdata/test-find-files-with-ext/"),
			".py",
			Want{[]string{}, nil},
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			sqlFiles, err := findFilesWithExt(testCase.fsys, testCase.ext)
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
	// Create test cases to test that the function only accepts pointers to structs
	var nilPtr *int = nil
	num := 1
	intPtr := &num
	testCases := []struct {
		v   any
		err error
	}{
		{
			1,
			fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries),
		},
		{
			"",
			fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries),
		},
		{
			struct{ CreateCatTable string }{},
			fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries),
		},
		{
			nil,
			fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries),
		},
		{
			map[string]string{},
			fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries),
		},
		{
			nilPtr,
			fmt.Errorf("%w: v is nil", ErrCannotLoadQueries),
		},
		{
			intPtr,
			fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries),
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d (v=%v)", i, testCase.v), func(t *testing.T) {
			err := loadQueriesIntoStruct(map[string]string{}, testCase.v, false)
			if fmt.Sprint(err) != fmt.Sprint(testCase.err) {
				t.Errorf("got %s, want %s", err, testCase.err)
				return
			}
		})
	}
	// Create a struct that does not use strings
	type InvalidCatQuery struct {
		CreateCatTable int `query:"CreateCatTable"`
	}
	invalidCatQuery := InvalidCatQuery{}
	err := loadQueriesIntoStruct(CatTestQueries, &invalidCatQuery, false)
	wantedErr := fmt.Errorf("%w: field %s cannot be changed or is not a string", ErrCannotLoadQueries, "CreateCatTable")
	if fmt.Sprint(err) != fmt.Sprint(wantedErr) {
		t.Errorf("got %s, want %s", err, wantedErr)
	}
	// Create a struct that has a query that the cat-queries.sql file do not
	type MissingCatQueries struct {
		DeleteCatById int `query:"DeleteCatById"`
	}
	missingCatQueries := MissingCatQueries{}
	err = loadQueriesIntoStruct(CatTestQueries, &missingCatQueries, false)
	wantedErr = fmt.Errorf("%w: could not find query %s", ErrCannotLoadQueries, "DeleteCatById")
	if fmt.Sprint(err) != fmt.Sprint(wantedErr) {
		t.Errorf("got %s, want %s", err, wantedErr)
	}
	// Create struct to hold the queries
	type CatQuery struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
		CreateNormalCat string `query:"CreateNormalCat"`
		UpdateColorById string `query:"UpdateColorById"`
	}
	catQuery := CatQuery{}
	err = loadQueriesIntoStruct(CatTestQueries, &catQuery, false)
	if err != nil {
		t.Fatalf("err must be nil, got %s", err)
	}
	if catQuery.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQuery.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if catQuery.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQuery.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
	}
	if catQuery.CreateNormalCat != CatTestQueries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", catQuery.CreateNormalCat, CatTestQueries["CreateNormalCat"])
	}
	if catQuery.UpdateColorById != CatTestQueries["UpdateColorById"] {
		t.Errorf("got %s, want %s", catQuery.UpdateColorById, CatTestQueries["UpdateColorById"])
	}
}

func TestCat(t *testing.T) {
	fsys := os.DirFS("testdata/test-cat")
	txt, err := cat(fsys, []string{"file1.txt", "file2.txt"})
	if err != nil {
		t.Fatalf("err must be nil, got %s", err)
	}
	wantedTxt := `Some text around here...

Even more text around there...
`
	if txt != wantedTxt {
		t.Fatalf("got %s, want %s", txt, wantedTxt)
	}
	fsys = os.DirFS("testdata/i-dont-exist")
	_, err = cat(fsys, []string{"i-dont-exist.sql"})
	if err == nil {
		t.Fatalf("err must not be nil")
	}
}

func TestLoadFromString(t *testing.T) {
	sql := `
	-- query: invalid-name
	`
	_, err := LoadFromString[struct{}](sql)
	want := fmt.Errorf("%w: invalid query name invalid-name", ErrCannotLoadQueries)
	if fmt.Sprint(err) != fmt.Sprint(want) {
		t.Fatalf("got %s, want %s", err, want)
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
	_, err = LoadFromString[int](sql)
	if err == nil {
		t.Fatal("err is nil")
	}
	q, err := LoadFromString[struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
	}](sql)
	if err != nil {
		t.Fatalf("err must be nil, got %s", err)
	}
	if q.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", q.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if q.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", q.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
	}
}

func TestMustLoadFromString(t *testing.T) {
	// Test that the function panics if any error occurs
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("function did not panic")
			}
		}()
		sql := `
		-- query: invalid-name
		`
		MustLoadFromString[struct{}](sql)
	}()
	// Test that the function does not panic if no errors occur
	sql := ""
	MustLoadFromString[struct{}](sql)
}

func TestFromFile(t *testing.T) {
	type CatQuery struct {
		CreateCatTable  string `query:"CreateCatTable"`
		CreatePsychoCat string `query:"CreatePsychoCat"`
		CreateNormalCat string `query:"CreateNormalCat"`
		UpdateColorById string `query:"UpdateColorById"`
	}
	_, err := LoadFromFile[CatQuery]("testdata/i-dont-exist.sql")
	if err == nil {
		t.Fatalf("file testdata/i-dont-exist.sql must not exists so this test can fail")
	}
	// test using LF line endings
	catQuery, err := LoadFromFile[CatQuery]("testdata/cat-queries.sql")
	if err != nil {
		t.Fatalf("error loading testdata/cat-queries.sql: %s", err)
	}
	if catQuery.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQuery.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if catQuery.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQuery.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
	}
	if catQuery.CreateNormalCat != CatTestQueries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", catQuery.CreateNormalCat, CatTestQueries["CreateNormalCat"])
	}
	if catQuery.UpdateColorById != CatTestQueries["UpdateColorById"] {
		t.Errorf("got %s, want %s", catQuery.UpdateColorById, CatTestQueries["UpdateColorById"])
	}
	// test using CRLF line endings
	catQuery, err = LoadFromFile[CatQuery]("testdata/cat-queries.crlf.sql")
	if err != nil {
		t.Fatalf("error loading testdata/cat-queries.sql: %s", err)
	}
	if catQuery.CreateCatTable != CatTestQueries["CreateCatTable"] {
		t.Errorf("got %s, want %s", catQuery.CreateCatTable, CatTestQueries["CreateCatTable"])
	}
	if catQuery.CreatePsychoCat != CatTestQueries["CreatePsychoCat"] {
		t.Errorf("got %s, want %s", catQuery.CreatePsychoCat, CatTestQueries["CreatePsychoCat"])
	}
	if catQuery.CreateNormalCat != CatTestQueries["CreateNormalCat"] {
		t.Errorf("got %s, want %s", catQuery.CreateNormalCat, CatTestQueries["CreateNormalCat"])
	}
	if catQuery.UpdateColorById != CatTestQueries["UpdateColorById"] {
		t.Errorf("got %s, want %s", catQuery.UpdateColorById, CatTestQueries["UpdateColorById"])
	}
}

func TestMustLoadFromFile(t *testing.T) {
	// Test that the function panics if any error occurs
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("function did not panic")
			}
		}()
		MustLoadFromFile[struct{}]("testdata/i-dont-exist.sql")
	}()
	// Test that the function does not panic if no errors occur
	MustLoadFromFile[struct{}]("testdata/cat-queries.sql")
}

func TestLoadFromDir(t *testing.T) {
	type RandomQuery struct {
		CreateCatTable      string `query:"CreateCatTable"`
		CreatePsychoCat     string `query:"CreatePsychoCat"`
		CreateNormalCat     string `query:"CreateNormalCat"`
		UpdateColorById     string `query:"UpdateColorById"`
		FindUserById        string `query:"FindUserById"`
		UpdateFirstNameById string `query:"UpdateFirstNameById"`
		DeleteUserById      string `query:"DeleteUserById"`
		FindRiders          string `query:"FindRiders"`
	}
	// Test that the function fails when the directory does not exist
	_, err := LoadFromDir[RandomQuery]("testdata/i-dont-exist")
	if err == nil {
		t.Fatalf("dir testdata/i-dont-exist must not exists so this test can fail")
	}

	// Permission-based tests do not work on Windows
	if runtime.GOOS != "windows" {
		// Test that the function fails when it can not read some .sql file
		unreadableFilename := "testdata/test-load-from-dir/unreadable-file.sql"
		unreadableFile, err := os.Create(unreadableFilename)
		if err != nil {
			t.Fatalf("unable to create %s: %s", unreadableFilename, err)
		}
		defer unreadableFile.Close()
		err = os.Chmod(unreadableFilename, 0222)
		if err != nil {
			t.Fatalf("unable to set the permissions of %s to 0222: %s", unreadableFilename, err)
		}
		_, err = LoadFromDir[RandomQuery]("testdata/test-load-from-dir")
		if err == nil {
			t.Fatal("error is nil")
		}
		err = os.Remove(unreadableFilename)
		if err != nil {
			t.Fatalf("unable to remove %s: %s", unreadableFilename, err)
		}
	}
	// Test that the function succeeds when using the happy path
	queries, err := LoadFromDir[RandomQuery]("testdata/test-load-from-dir")
	if err != nil {
		t.Fatalf("error loading testdata/test-load-from-dir: %s", err)
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

func TestMustLoadFromDir(t *testing.T) {
	// Test that the function panics if any error occurs
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("function did not panic")
			}
		}()
		MustLoadFromDir[struct{}]("testdata/i-dont-exist")
	}()
	// Test that the function does not panic if no errors occur
	MustLoadFromDir[struct{}]("testdata/test-load-from-dir")
}

func TestLoadFromFS(t *testing.T) {
	type RandomQuery struct {
		CreateCatTable      string `query:"CreateCatTable"`
		CreatePsychoCat     string `query:"CreatePsychoCat"`
		CreateNormalCat     string `query:"CreateNormalCat"`
		UpdateColorById     string `query:"UpdateColorById"`
		FindUserById        string `query:"FindUserById"`
		UpdateFirstNameById string `query:"UpdateFirstNameById"`
		DeleteUserById      string `query:"DeleteUserById"`
		FindRiders          string `query:"FindRiders"`
	}
	// Test that the function fails when the directory does not exist
	fsys := os.DirFS("testdata/i-dont-exist")
	_, err := LoadFromFS[RandomQuery](fsys)
	if err == nil {
		t.Fatalf("dir testdata/i-dont-exist must not exists so this test can fail")
	}
	// Permission-based tests do not work on Windows
	if runtime.GOOS != "windows" {
		// Test that the function fails when it can not read some .sql file
		unreadableFilename := "testdata/test-load-from-fs/unreadable-file.sql"
		unreadableFile, err := os.Create(unreadableFilename)
		if err != nil {
			t.Fatalf("unable to create %s: %s", unreadableFilename, err)
		}
		defer unreadableFile.Close()
		err = os.Chmod(unreadableFilename, 0222)
		if err != nil {
			t.Fatalf("unable to set the permissions of %s to 0222: %s", unreadableFilename, err)
		}
		fsys = os.DirFS("testdata/test-load-from-fs")
		_, err = LoadFromFS[RandomQuery](fsys)
		if err == nil {
			t.Fatal("error is nil")
		}
		err = os.Remove(unreadableFilename)
		if err != nil {
			t.Fatalf("unable to remove %s: %s", unreadableFilename, err)
		}
	}
	// Test that the function succeeds when using the happy path
	fsys = os.DirFS("testdata/test-load-from-fs")
	queries, err := LoadFromFS[RandomQuery](fsys)
	if err != nil {
		t.Fatalf("error loading testdata/test-load-from-fs: %s", err)
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

func TestMustLoadFromFS(t *testing.T) {
	// Test that the function panics if any error occurs
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("function did not panic")
			}
		}()
		fsys := os.DirFS("testdata/i-dont-exist")
		MustLoadFromFS[struct{}](fsys)
	}()
	// Test that the function does not panic if no errors occur
	fsys := os.DirFS("testdata/test-load-from-fs")
	MustLoadFromFS[struct{}](fsys)
}
