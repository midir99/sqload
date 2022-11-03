# sqload

Personally I don't like writing SQL code inside the Go source files, so I made this simple and fully tested module to load SQL queries from files.

## How to use it?

Each SQL query must include a comment at the beginning, the comment must be something like:

`-- query: NameOfYourQuery`

This comment is mandatory so the loader can match the name of your query with the field of the struct where the SQL code of your query will be stored. In this case, the struct would look like this:

```go
type Query struct {
    NameOfYourQuery string `query:"NameOfYourQuery"`
}
```

The following are some examples of how you can use this library:

### Load SQL code from strings

```go
package main

import (
	"fmt"
	"os"

	"github.com/midir99/sqload"
)

type UserQuery struct {
	FindUserById            string `query:"FindUserById"`
	UpdateUserFirstNameById string `query:"UpdateUserFirstNameById"`
}

func main() {
	sql := `
	-- query: FindUserById
	SELECT * FROM user WHERE id = :id;

	-- query: UpdateUserFirstNameById
	UPDATE user SET first_name = :first_name WHERE id = :id;
	`
	userQuery := UserQuery{}
	err := sqload.FromString(sql, &userQuery)
	if err != nil {
		fmt.Printf("error loading user queries: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("FindUserById: %s\n", userQuery.FindUserById)
	fmt.Printf("UpdateUserFirstNameById: %s\n", userQuery.UpdateUserFirstNameById)
}
```

### Load SQL code from files

`file: cat-queries.sql`
```sql
-- query: CreateCatTable
CREATE TABLE Cat (
    id SERIAL,
    name VARCHAR(150),
    color VARCHAR(50),
    PRIMARY KEY (id)
);

-- query: CreatePsychoCat
INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');

-- query: CreateNormalCat
INSERT INTO Cat (name, color) VALUES (:name, :color);

-- query: UpdateColorById
UPDATE Cat
   SET color = :color
 WHERE id = :id;
```

`file: main.go`
```go
package main

import (
	"fmt"
	"os"

	"github.com/midir99/sqload"
)

type CatQuery struct {
	CreateCatTable  string `query:"CreateCatTable"`
	CreatePsychoCat string `query:"CreatePsychoCat"`
	CreateNormalCat string `query:"CreateNormalCat"`
	UpdateColorById string `query:"UpdateColorById"`
}

func main() {
	catQuery := CatQuery{}
	err := sqload.FromFile("cat-queries.sql", &catQuery)
	if err != nil {
		fmt.Printf("error loading cat queries: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("CreateCatTable: %s\n", catQuery.CreateCatTable)
	fmt.Printf("CreatePsychoCat: %s\n", catQuery.CreatePsychoCat)
	fmt.Printf("CreateNormalCat: %s\n", catQuery.CreateNormalCat)
	fmt.Printf("UpdateColorById: %s\n", catQuery.UpdateColorById)
}
```

### Load SQL code from directories containing .sql files

Lets say you have a directory containing your SQL files:
```
sql/
├── cat-queries.sql
└── user-queries.sql
```

`file: cat-queries.sql`
```sql
-- query: CreateCatTable
CREATE TABLE Cat (
    id SERIAL,
    name VARCHAR(150),
    color VARCHAR(50),
    PRIMARY KEY (id)
);

-- query: CreatePsychoCat
INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');

-- query: CreateNormalCat
INSERT INTO Cat (name, color) VALUES (:name, :color);

-- query: UpdateColorById
UPDATE Cat
   SET color = :color
 WHERE id = :id;
```

`file: user-queries.sql`
```sql
-- query: FindUserById
SELECT * FROM user WHERE id = :id;

-- query: UpdateUserFirstNameById
UPDATE user SET first_name = :first_name WHERE id = :id;
```

`file: main.go`

```go
package main

import (
	"fmt"
	"os"

	"github.com/midir99/sqload"
)

type Query struct {
	CreateCatTable  string `query:"CreateCatTable"`
	CreatePsychoCat string `query:"CreatePsychoCat"`
	CreateNormalCat string `query:"CreateNormalCat"`
	UpdateColorById string `query:"UpdateColorById"`

	FindUserById            string `query:"FindUserById"`
	UpdateUserFirstNameById string `query:"UpdateUserFirstNameById"`
}

func main() {
	query := Query{}
	err := sqload.FromDir("sql/", &query)
	if err != nil {
		fmt.Printf("error loading queries: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("CreateCatTable: %s\n", query.CreateCatTable)
	fmt.Printf("CreatePsychoCat: %s\n", query.CreatePsychoCat)
	fmt.Printf("CreateNormalCat: %s\n", query.CreateNormalCat)
	fmt.Printf("UpdateColorById: %s\n", query.UpdateColorById)

	fmt.Printf("FindUserById: %s\n", query.FindUserById)
	fmt.Printf("UpdateUserFirstNameById: %s\n", query.UpdateUserFirstNameById)
}
```
