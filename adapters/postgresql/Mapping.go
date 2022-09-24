package postgresql

import (
	"context"

	"github.com/adamluzsi/frameless/pkg/iterators"
)

type Mapping[Ent any] interface {
	// TableRef is the entity's postgresql table name.
	//   eg.:
	//     - "public"."table_name"
	//     - "table_name"
	//     - table_name
	//
	TableRef() string
	// IDRef is the entity's id column name, which can be used to access an individual record for update purpose.
	IDRef() string
	// ColumnRefs are the table's column names.
	// The order of the column names related to Row mapping and query argument passing.
	ColumnRefs() []string
	// NewID creates a stateless entity id that can be used by CREATE operation.
	// Serial and similar id solutions not supported without serialize transactions.
	NewID(context.Context) (interface{}, error)
	// ToArgs convert an entity ptr to a list of query argument that can be used for CREATE or UPDATE purpose.
	ToArgs(ptr *Ent) ([]interface{}, error)
	iterators.SQLRowMapper[Ent]
}