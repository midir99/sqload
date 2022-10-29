package sqload

import "testing"

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
				"-- Find all users",
				"SELECT *",
				"FROM user;",
			},
			"SELECT * FROM user;",
		},
	}
}

func TestExtractQueries(t *testing.T) {

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
