package queries

import (
	"github.com/adamluzsi/frameless/reflects"
	"github.com/adamluzsi/frameless/resources"
	"testing"

	"github.com/adamluzsi/frameless/fixtures"

	"github.com/adamluzsi/frameless"
	"github.com/stretchr/testify/require"
)

// DeleteEntity request a destroy of a specific entity that is wrapped in the query use case object
type DeleteEntity struct {
	Entity frameless.Entity
}

// Test will test that an DeleteEntity is implemented by a generic specification
func (quc DeleteEntity) Test(spec *testing.T, r frameless.Resource) {

	spec.Run("dependency", func(t *testing.T) {
		SaveEntity{Entity: quc.Entity}.Test(t, r)
	})

	expected := fixtures.New(quc.Entity)
	require.Nil(spec, r.Exec(SaveEntity{Entity: expected}).Err())
	ID, ok := resources.LookupID(expected)

	if !ok {
		spec.Fatal(ErrIDRequired)
	}

	defer r.Exec(DeleteByID{Type: reflects.BaseValueOf(quc.Entity).Interface(), ID: ID})

	spec.Run("value is Deleted by providing an Entity, and then it should not be findable afterwards", func(t *testing.T) {

		deleteResults := r.Exec(DeleteEntity{Entity: expected})
		require.NotNil(t, deleteResults)
		require.Nil(t, deleteResults.Err())

		iterator := r.Exec(FindByID{Type: reflects.BaseValueOf(quc.Entity).Interface(), ID: ID})
		defer iterator.Close()

		if iterator.Next() {
			var entity frameless.Entity
			iterator.Decode(&entity)
			t.Fatalf("there should be no next value, but %#v found", entity)
		}

	})

	spec.Run("when entity doesn't have r ID field", func(t *testing.T) {
		newEntity := fixtures.New(entityWithoutIDField{})
		require.Error(t, r.Exec(DeleteEntity{Entity: newEntity}).Err())
	})
}