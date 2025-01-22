// Package sqload provides functions to load SQL code from strings or .sql files into
// tagged struct fields.
//
// Its usage is very straightforward; let's suppose you have the following SQL file:
//
// File queries.sql:
//
//	-- query: FindUserById
//	-- Finds a user by its id.
//	SELECT first_name,
//	       last_name,
//	       dob,
//	       email
//	  FROM user
//	 WHERE id = :id;
//
//	-- query: UpdateFirstNameById
//	UPDATE user
//	   SET first_name = 'Ernesto'
//	 WHERE id = :id;
//
//	-- query: DeleteUserById
//	-- Deletes a user by its id.
//	DELETE FROM user
//	      WHERE id = :id;
//
// You could load the SQL code of those queries into strings using the following:
//
// File main.go:
//
//	package main
//
//	import (
//		_ "embed"
//		"fmt"
//
//		"github.com/midir99/sqload"
//	)
//
//	//go:embed queries.sql
//	var sqlCode string
//
//	var Q = sqload.MustLoadFromString[struct {
//		FindUserById        string `query:"FindUserById"`
//		UpdateFirstNameById string `query:"UpdateFirstNameById"`
//		DeleteUserById      string `query:"DeleteUserById"`
//	}](sqlCode)
//
//	func main() {
//		fmt.Printf("- FindUserById\n%s\n\n", Q.FindUserById)
//		fmt.Printf("- UpdateFirstNameById\n%s\n\n", Q.UpdateFirstNameById)
//		fmt.Printf("- DeleteUserById\n%s\n\n", Q.DeleteUserById)
//	}
//
// The module maps the fields of your struct and the queries from the SQL file using the
// query tag (in the struct):
//
//	`query:"NameOfYourQuery"`
//
// And the query comment (in the SQL code):
//
//	-- query: NameOfYourQuery
package sqload

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

// Struct is an empty interface used to give the developer a hint that the type must be
// a struct.
type Struct interface{}

var ErrCannotLoadQueries = errors.New("cannot load queries")

var queryNamePattern = regexp.MustCompile(`[ \t\n\r\f\v]*-- query:`)
var validQueryNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
var queryCommentPattern = regexp.MustCompile(`[ \t\n\r\f\v]*--[ \t\n\r\f\v]*(.*)$`)
var newLinePattern = regexp.MustCompile("\r?\n")

func extractSql(lines []string) string {
	sqlLines := []string{}
	for _, line := range lines {
		if !queryCommentPattern.MatchString(line) {
			sqlLines = append(sqlLines, line)
		}
	}
	return strings.Join(sqlLines, "\n")
}

// ExtractQueryMap extracts the SQL code from the string and returns a map containing the queries.
// The query name is the key in each map entry, and the SQL code is its value.
//
//	package main
//
//	import (
//	        "fmt"
//	        "os"
//
//	        "github.com/midir99/sqload"
//	)
//
//	func main() {
//	        q, err := sqload.ExtractQueryMap(`
//	-- query: FindUserById
//	SELECT first_name,
//	       last_name,
//	       dob,
//	       email
//	  FROM user
//	 WHERE id = :id;
//
//	-- query: DeleteUserById
//	DELETE FROM user
//	      WHERE id = :id;
//	        `)
//	        if err != nil {
//	                fmt.Printf("Unable to load SQL queries: %s\n", err)
//	                os.Exit(1)
//	        }
//	        if findUserById, found := q["FindUserById"]; found {
//	                fmt.Printf("- FindUserById\n%s\n\n", findUserById)
//	        }
//	        for k, v := range q {
//	                fmt.Printf("- %s\n%s\n\n", k, v)
//	        }
//	}
func ExtractQueryMap(sql string) (map[string]string, error) {
	queries := make(map[string]string)
	rawQueries := queryNamePattern.Split(sql, -1)
	if len(rawQueries) <= 1 {
		return queries, nil
	}
	for _, q := range rawQueries[1:] {
		lines := newLinePattern.Split(strings.TrimSpace(q), -1)
		queryName := lines[0]
		if !validQueryNamePattern.MatchString(queryName) {
			return nil, fmt.Errorf("%w: invalid query name %s", ErrCannotLoadQueries, queryName)
		}
		querySql := extractSql(lines[1:])
		queries[queryName] = querySql
	}
	return queries, nil
}

func findFilesWithExt(fsys fs.FS, ext string) ([]string, error) {
	files := []string{}
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%w: %w", ErrCannotLoadQueries, err)
		}
		if !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ext {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func loadQueriesIntoStruct(queries map[string]string, v Struct) error {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Pointer {
		return fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries)
	}
	if value.IsNil() {
		return fmt.Errorf("%w: v is nil", ErrCannotLoadQueries)
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("%w: v is not a pointer to a struct", ErrCannotLoadQueries)
	}
	queriesAndFields := map[string]int{}
	for i := 0; i < elem.NumField(); i++ {
		queryTag := elem.Type().Field(i).Tag.Get("query")
		if queryTag != "" {
			queriesAndFields[queryTag] = i
		}
	}
	for queryName, fieldIndex := range queriesAndFields {
		sql, ok := queries[queryName]
		if !ok {
			return fmt.Errorf("%w: could not find query %s", ErrCannotLoadQueries, queryName)
		}
		field := elem.Field(fieldIndex)
		if !field.CanSet() || field.Kind() != reflect.String {
			return fmt.Errorf("%w: field %s cannot be changed or is not a string", ErrCannotLoadQueries, elem.Type().Field(fieldIndex).Name)
		}
		field.SetString(sql)
	}
	return nil
}

func cat(fsys fs.FS, filenames []string) (string, error) {
	lines := []string{}
	for _, filename := range filenames {
		data, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrCannotLoadQueries, err)
		}
		lines = append(lines, string(data))
	}
	txt := strings.Join(lines, "\n")
	return txt, nil
}

// LoadFromString loads the SQL code from the string and returns a pointer to a struct.
// Each struct field will contain the SQL query code it was tagged with.
//
// If some query has an invalid name in the string or is not found in the string, it
// will return a nil pointer and an error.
//
//	package main
//
//	import (
//		"fmt"
//		"os"
//
//		"github.com/midir99/sqload"
//	)
//
//	func main() {
//		q, err := sqload.LoadFromString[struct {
//			FindUserById        string `query:"FindUserById"`
//			UpdateFirstNameById string `query:"UpdateFirstNameById"`
//			DeleteUserById      string `query:"DeleteUserById"`
//		}](`
//	-- query: FindUserById
//	SELECT first_name,
//	       last_name,
//	       dob,
//	       email
//	  FROM user
//	 WHERE id = :id;
//
//	-- query: UpdateFirstNameById
//	UPDATE user
//	   SET first_name = 'Ernesto'
//	 WHERE id = :id;
//
//	-- query: DeleteUserById
//	DELETE FROM user
//	      WHERE id = :id;
//		`)
//		if err != nil {
//			fmt.Printf("Unable to load SQL queries: %s\n", err)
//			os.Exit(1)
//		}
//		fmt.Printf("- FindUserById\n%s\n\n", q.FindUserById)
//		fmt.Printf("- UpdateFirstNameById\n%s\n\n", q.UpdateFirstNameById)
//		fmt.Printf("- DeleteUserById\n%s\n\n", q.DeleteUserById)
//	}
func LoadFromString[V Struct](s string) (*V, error) {
	var v V
	queries, err := ExtractQueryMap(s)
	if err != nil {
		return nil, err
	}
	err = loadQueriesIntoStruct(queries, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// MustLoadFromString is like LoadFromString but panics if any error occurs. It
// simplifies the safe initialization of global variables holding struct pointers
// containing SQL queries.
func MustLoadFromString[V Struct](s string) *V {
	v, err := LoadFromString[V](s)
	if err != nil {
		panic(err)
	}
	return v
}

// LoadFromFile loads the SQL code from the file filename and returns a pointer to a
// struct. Each struct field will contain the SQL query code it was tagged with.
//
// If some query has an invalid name in the string or is not found in the string, it
// will return a nil pointer and an error.
//
// If the file can not be read or does not exist, it will return a nil pointer and an
// error.
//
// File queries.sql:
//
//	-- query: FindUserById
//	SELECT first_name,
//	       last_name,
//	       dob,
//	       email
//	  FROM user
//	 WHERE id = :id;
//
//	-- query: UpdateFirstNameById
//	UPDATE user
//	   SET first_name = 'Ernesto'
//	 WHERE id = :id;
//
//	-- query: DeleteUserById
//	DELETE FROM user
//	      WHERE id = :id;
//
// File main.go:
//
//	package main
//
//	import (
//		"fmt"
//		"os"
//
//		"github.com/midir99/sqload"
//	)
//
//	func main() {
//		q, err := sqload.LoadFromFile[struct {
//			FindUserById        string `query:"FindUserById"`
//			UpdateFirstNameById string `query:"UpdateFirstNameById"`
//			DeleteUserById      string `query:"DeleteUserById"`
//		}]("queries.sql")
//		if err != nil {
//			fmt.Printf("Unable to load SQL queries: %s\n", err)
//			os.Exit(1)
//		}
//		fmt.Printf("- FindUserById\n%s\n\n", q.FindUserById)
//		fmt.Printf("- UpdateFirstNameById\n%s\n\n", q.UpdateFirstNameById)
//		fmt.Printf("- DeleteUserById\n%s\n\n", q.DeleteUserById)
//	}
func LoadFromFile[V Struct](filename string) (*V, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotLoadQueries, err)
	}
	return LoadFromString[V](string(data))
}

// MustLoadFromFile is like LoadFromFile but panics if any error occurs. It simplifies
// the safe initialization of global variables holding struct pointers containing SQL
// queries.
func MustLoadFromFile[V Struct](filename string) *V {
	v, err := LoadFromFile[V](filename)
	if err != nil {
		panic(err)
	}
	return v
}

// LoadFromDir loads the SQL code from all the .sql files in the directory dirname
// (recursively) and returns a pointer to a struct. Each struct field will contain the
// SQL query code it was tagged with.
//
// If some query has an invalid name in the string or is not found in the string, it
// will return a nil pointer and an error.
//
// If the directory can not be read or does not exist, it will return a nil pointer and
// an error.
//
// If any .sql file can not be read, it will return a nil pointer and an error.
//
// Project directory:
//
//	.
//	├── go.mod
//	├── main.go
//	└── sql
//	    ├── cats.sql
//	    └── users.sql
//
// File sql/cats.sql:
//
//	-- query: CreatePsychoCat
//	INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');
//
// File sql/users.sql:
//
//	-- query: DeleteUserById
//	DELETE FROM user WHERE id = :id;
//
// File main.go:
//
//	package main
//
//	import (
//		"fmt"
//		"os"
//
//		"github.com/midir99/sqload"
//	)
//
//	func main() {
//		q, err := sqload.LoadFromDir[struct {
//			CreatePsychoCat string `query:"CreatePsychoCat"`
//			DeleteUserById  string `query:"DeleteUserById"`
//		}]("sql")
//		if err != nil {
//			fmt.Printf("Unable to load SQL queries: %s\n", err)
//			os.Exit(1)
//		}
//		fmt.Printf("- CreatePsychoCat\n%s\n\n", q.CreatePsychoCat)
//		fmt.Printf("- DeleteUserById\n%s\n\n", q.DeleteUserById)
//	}
func LoadFromDir[V Struct](dirname string) (*V, error) {
	fsys := os.DirFS(dirname)
	files, err := findFilesWithExt(fsys, ".sql")
	if err != nil {
		return nil, err
	}
	sql, err := cat(fsys, files)
	if err != nil {
		return nil, err
	}
	return LoadFromString[V](sql)
}

// MustLoadFromDir is like LoadFromDir but panics if any error occurs. It simplifies the
// safe initialization of global variables holding struct pointers containing SQL
// queries.
func MustLoadFromDir[V Struct](dirname string) *V {
	v, err := LoadFromDir[V](dirname)
	if err != nil {
		panic(err)
	}
	return v
}

// LoadFromFS loads the SQL code from all the .sql files in the fsys file system
// (recursively) and returns a pointer to a struct. Each struct field will contain the
// SQL query code it was tagged with.
//
// If some query has an invalid name in the string or is not found in the string, it
// will return a nil pointer and an error.
//
// If the fsys can not be read or does not exist, it will return a nil pointer and
// an error.
//
// If any .sql file can not be read, it will return a nil pointer and an error.
//
// Project directory:
//
//	.
//	├── go.mod
//	├── main.go
//	└── sql
//	    ├── cats.sql
//	    └── users.sql
//
// File sql/cats.sql:
//
//	-- query: CreatePsychoCat
//	INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');
//
// File sql/users.sql:
//
//	-- query: DeleteUserById
//	DELETE FROM user WHERE id = :id;
//
// File main.go:
//
//	package main
//
//	import (
//		"embed"
//		"fmt"
//		"os"
//
//		"github.com/midir99/sqload"
//	)
//
//	//go:embed sql/*.sql
//	var fsys embed.FS
//
//	func main() {
//		q, err := sqload.LoadFromFS[struct {
//			CreatePsychoCat string `query:"CreatePsychoCat"`
//			DeleteUserById  string `query:"DeleteUserById"`
//		}](fsys)
//		if err != nil {
//			fmt.Printf("Unable to load SQL queries: %s\n", err)
//			os.Exit(1)
//		}
//		fmt.Printf("- CreatePsychoCat\n%s\n\n", q.CreatePsychoCat)
//		fmt.Printf("- DeleteUserById\n%s\n\n", q.DeleteUserById)
//	}
func LoadFromFS[V Struct](fsys fs.FS) (*V, error) {
	files, err := findFilesWithExt(fsys, ".sql")
	if err != nil {
		return nil, err
	}
	sql, err := cat(fsys, files)
	if err != nil {
		return nil, err
	}
	return LoadFromString[V](sql)
}

// MustLoadFromFS is like LoadFromFS but panics if any error occurs. It simplifies the
// safe initialization of global variables holding struct pointers containing SQL
// queries.
func MustLoadFromFS[V Struct](fsys fs.FS) *V {
	v, err := LoadFromFS[V](fsys)
	if err != nil {
		panic(err)
	}
	return v
}
