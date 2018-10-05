package update

import (
	"github.com/adamluzsi/frameless/iterators"
	"github.com/adamluzsi/frameless/queries/find"
	"github.com/adamluzsi/frameless/queries/fixtures"
	"github.com/adamluzsi/frameless/queries/save"
	"github.com/adamluzsi/frameless/queries/queryerrors"
	"github.com/adamluzsi/frameless/storages"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/adamluzsi/frameless"
)

// ByEntity will request an update for a wrapped entity object in the storage
// ByEntity parameter is the wrapped entity that has the updated values.
type ByEntity struct{ Entity frameless.Entity }

func (quc ByEntity) Test(suite *testing.T, storage frameless.Storage, reset func()) {
	suite.Run("ByEntity", func(spec *testing.T) {

		suite.Run("dependency", func(t *testing.T) {
			save.Entity{Entity: quc.Entity}.Test(t, storage, reset)
		})

		setup := func() (string, func()) {
			entity := fixtures.New(quc.Entity)
			require.Nil(spec, storage.Exec(save.Entity{Entity: entity}).Err())

			ID, ok := storages.LookupID(entity)

			if !ok {
				spec.Fatal(queryerrors.ErrIDRequired)
			}

			require.True(spec, len(ID) > 0)

			return ID, reset
		}

		spec.Run("values returned", func(t *testing.T) {
			ID, td := setup()
			defer td()

			newEntity := fixtures.New(quc.Entity)
			storages.SetID(newEntity, ID)

			updateResults := storage.Exec(ByEntity{Entity: newEntity})
			require.NotNil(t, updateResults)
			require.Nil(t, updateResults.Err())

			iterator := storage.Exec(find.ByID{Type: quc.Entity, ID: ID})

			actually := fixtures.New(quc.Entity)
			iterators.DecodeNext(iterator, actually)

			require.Equal(t, newEntity, actually)

		})

		spec.Run("values in the storage but the requested entity that should be updated is not exists", func(t *testing.T) {
			_, td := setup()
			defer td()

			newEntity := fixtures.New(quc.Entity)
			storages.SetID(newEntity, "hitchhiker's guide to the galaxy")
			require.Error(t, storage.Exec(ByEntity{Entity: newEntity}).Err())

		})

	})
}
