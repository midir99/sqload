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

var QueryName = regexp.MustCompile(`[ \t\n\r\f\v]*-- name:`)
var ValidQueryName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
var QueryComment = regexp.MustCompile(`[ \t\n\r\f\v]*--[ \t\n\r\f\v]*(.*)$`)

func extractSql(lines []string) string {
	sqlLines := []string{}
	for _, line := range lines {
		if !QueryComment.MatchString(line) {
			sqlLines = append(sqlLines, line)
		}
	}
	return strings.Join(sqlLines, "\n")
}

func extractQueries(sql string) (map[string]string, error) {
	queries := make(map[string]string)
	rawQueries := QueryName.Split(sql, -1)
	if len(rawQueries) <= 1 {
		return queries, nil
	}
	for _, q := range rawQueries[1:] {
		lines := strings.Split(strings.TrimSpace(q), "\n")
		queryName := lines[0]
		if !ValidQueryName.MatchString(queryName) {
			return nil, fmt.Errorf("invalid query name: %s", queryName)
		}
		querySql := extractSql(lines[1:])
		queries[queryName] = querySql
	}
	return queries, nil
}

func findFilesWithExtension(dir string, ext string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
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
		return fmt.Errorf("v is not a pointer")
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

func FromString(s string, v any) error {
	queries, err := extractQueries(s)
	if err != nil {
		return err
	}
	return loadQueriesIntoStruct(queries, v)
}

func FromFile(name string, v any) error {
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	return FromString(string(data), v)
}

func FromDir(name string, v any) error {
	files, err := findFilesWithExtension(name, ".sql")
	if err != nil {
		return err
	}
	queries := map[string]string{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		moreQueries, err := extractQueries(string(data))
		if err != nil {
			return err
		}
		for k, v := range moreQueries {
			queries[k] = v
		}
	}
	err = loadQueriesIntoStruct(queries, v)
	if err != nil {
		return err
	}
	return nil
}
