// Package sqload provides functions to load SQL code from strings or .sql files into tagged struct fields.
package sqload

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

var queryNamePattern = regexp.MustCompile(`[ \t\n\r\f\v]*-- query:`)
var validQueryNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
var queryCommentPattern = regexp.MustCompile(`[ \t\n\r\f\v]*--[ \t\n\r\f\v]*(.*)$`)

func extractSql(lines []string) string {
	sqlLines := []string{}
	for _, line := range lines {
		if !queryCommentPattern.MatchString(line) {
			sqlLines = append(sqlLines, line)
		}
	}
	return strings.Join(sqlLines, "\n")
}

func extractQueries(sql string) (map[string]string, error) {
	queries := make(map[string]string)
	rawQueries := queryNamePattern.Split(sql, -1)
	if len(rawQueries) <= 1 {
		return queries, nil
	}
	for _, q := range rawQueries[1:] {
		lines := strings.Split(strings.TrimSpace(q), "\n")
		queryName := lines[0]
		if !validQueryNamePattern.MatchString(queryName) {
			return nil, fmt.Errorf("invalid query name: %s", queryName)
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
			return err
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

func loadQueriesIntoStruct(queries map[string]string, v any) error {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Pointer {
		return fmt.Errorf("v is not a pointer to a struct")
	}
	if value.IsNil() {
		return fmt.Errorf("v is nil")
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("v is not a pointer to a struct")
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
			return fmt.Errorf("could not to find query %s", queryName)
		}
		field := elem.Field(fieldIndex)
		if !field.CanSet() || field.Kind() != reflect.String {
			return fmt.Errorf("field %s cannot be changed or is not a string", elem.Type().Field(fieldIndex).Name)
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
			return "", err
		}
		lines = append(lines, string(data))
	}
	txt := strings.Join(lines, "\n")
	return txt, nil
}

// LoadFromString loads the SQL code from the string passed and stores the queries in the
// struct pointed to by v, v must be a pointer to a struct with tags, and each tag
// indicates what query will be stored in what field.
func LoadFromString[V any](s string) (*V, error) {
	var v V
	queries, err := extractQueries(s)
	if err != nil {
		return nil, err
	}
	err = loadQueriesIntoStruct(queries, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func MustLoadFromString[V any](s string) *V {
	v, err := LoadFromString[V](s)
	if err != nil {
		panic(err)
	}
	return v
}

// LoadFromFile loads the SQL code from the file filename and stores the queries in the struct
// pointed to by v, v must be a pointer to a struct with tags, and each tag indicates
// what query will be stored in what field.
func LoadFromFile[V any](filename string) (*V, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return LoadFromString[V](string(data))
}

func MustLoadFromFile[V any](filename string) *V {
	v, err := LoadFromFile[V](filename)
	if err != nil {
		panic(err)
	}
	return v
}

// LoadFromDir loads the SQL code from all the .sql files in the directory dirname
// (recursively) and stores the queries in the struct pointed to by v, v must be a
// pointer to a struct with tags, and each tag indicates what query will be stored in
// what field.
func LoadFromDir[V any](dirname string) (*V, error) {
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

func MustLoadFromDir[V any](dirname string) *V {
	v, err := LoadFromDir[V](dirname)
	if err != nil {
		panic(err)
	}
	return v
}

func LoadFromFS[V any](fsys fs.FS) (*V, error) {
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

func MustLoadFromFS[V any](fsys fs.FS) *V {
	v, err := LoadFromFS[V](fsys)
	if err != nil {
		panic(err)
	}
	return v
}
