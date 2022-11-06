# sqload

https://pkg.go.dev/github.com/midir99/sqload

Personally, I don't like writing SQL code inside the Go source files, so I made this simple and thoroughly tested module to load SQL queries from files.

This library is inspired by [Yesql](https://github.com/krisajenkins/yesql/).

## How to use it?

Each SQL query must include a comment at the beginning; the comment must be something like:

```sql
-- query: FindCatById
SELECT * FROM cat WHERE id = :id;
```

This comment is mandatory so the loader can match the name of your query with the tagged struct field where the SQL code of your query will be stored. In this case, the struct would look like this:

```go
struct {
    FindCatById string `query:"FindCatById"`
}
```

### Load SQL code from strings

```go
package main

import (
	"fmt"

	"github.com/midir99/sqload"
)

var Q = sqload.MustLoadFromString[struct {
	FindUserById        string `query:"FindUserById"`
	UpdateFirstNameById string `query:"UpdateFirstNameById"`
	DeleteUserById      string `query:"DeleteUserById"`
}](`
-- query: FindUserById
SELECT first_name,
       last_name,
       dob,
       email
  FROM user
 WHERE id = :id;

-- query: UpdateFirstNameById
UPDATE user
   SET first_name = 'Ernesto'
 WHERE id = :id;

-- query: DeleteUserById
DELETE FROM user
      WHERE id = :id;
`)

func main() {
	fmt.Printf("- FindUserById\n%s\n\n", Q.FindUserById)
	fmt.Printf("- UpdateFirstNameById\n%s\n\n", Q.UpdateFirstNameById)
	fmt.Printf("- DeleteUserById\n%s\n\n", Q.DeleteUserById)
}
```

### Load SQL code from files using `embed`

Using the module `embed` to load your SQL files into strings and then passing those to `sqload` functions is a convenient approach.

`file queries.sql:`
```sql
-- query: FindUserById
SELECT first_name,
       last_name,
       dob,
       email
  FROM user
 WHERE id = :id;

-- query: UpdateFirstNameById
UPDATE user
   SET first_name = 'Ernesto'
 WHERE id = :id;

-- query: DeleteUserById
DELETE FROM user
      WHERE id = :id;
```

`file main.go:`
```go
package main

import (
	_ "embed"
	"fmt"

	"github.com/midir99/sqload"
)

//go:embed queries.sql
var sqlCode string

var Q = sqload.MustLoadFromString[struct {
	FindUserById        string `query:"FindUserById"`
	UpdateFirstNameById string `query:"UpdateFirstNameById"`
	DeleteUserById      string `query:"DeleteUserById"`
}](sqlCode)

func main() {
	fmt.Printf("- FindUserById\n%s\n\n", Q.FindUserById)
	fmt.Printf("- UpdateFirstNameById\n%s\n\n", Q.UpdateFirstNameById)
	fmt.Printf("- DeleteUserById\n%s\n\n", Q.DeleteUserById)
}
```

### Load SQL code from directories containing .sql files using `embed`

Lets say you have a directory containing your SQL files:
```
.
├── go.mod
├── main.go
└── sql           <- this one
    ├── cats.sql
    └── users.sql
```

`File sql/cats.sql:`
```sql
-- query: CreatePsychoCat
INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');
```

`File sql/users.sql:`
```sql
-- query: DeleteUserById
DELETE FROM user WHERE id = :id;
```

`File main.go:`

```go
package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/midir99/sqload"
)

//go:embed sql
var fsys embed.FS

func main() {
	q, err := sqload.LoadFromFS[struct {
		CreatePsychoCat string `query:"CreatePsychoCat"`
		DeleteUserById  string `query:"DeleteUserById"`
	}](fsys)
	if err != nil {
		fmt.Printf("Unable to load SQL queries: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("- CreatePsychoCat\n%s\n\n", q.CreatePsychoCat)
	fmt.Printf("- DeleteUserById\n%s\n\n", q.DeleteUserById)
}
```

Check more examples in the official documentation: https://pkg.go.dev/github.com/midir99/sqload
