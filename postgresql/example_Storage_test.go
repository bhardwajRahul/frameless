package postgresql_test

import (
	"context"
	"math/rand"
	"os"

	"github.com/adamluzsi/frameless/iterators"
	"github.com/adamluzsi/frameless/postgresql"
)

func ExampleStorage() {
	type Entity struct {
		ID    int `ext:"ID"`
		Value string
	}

	mapping := postgresql.Mapper{
		Table:   "entities",
		ID:      "id",
		Columns: []string{`id`, `value`},
		NewIDFn: func(ctx context.Context) (interface{}, error) {
			// only example, don't do this in production code.
			return rand.Int(), nil
		},
		ToArgsFn: func(ptr interface{}) ([]interface{}, error) {
			ent := ptr.(*Entity)
			return []interface{}{ent.ID, ent.Value}, nil
		},
		MapFn: func(s iterators.SQLRowScanner, ptr interface{}) error {
			ent := ptr.(*Entity)
			return s.Scan(&ent.ID, &ent.Value)
		},
	}

	stg, err := postgresql.NewStorageByDSN(Entity{}, mapping, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer stg.Close()
}
