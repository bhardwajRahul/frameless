package frameless_test

import (
	"fmt"
	"github.com/adamluzsi/frameless/queries"
	"github.com/adamluzsi/frameless/reflects"
	"io"
	"testing"

	"github.com/adamluzsi/frameless/iterators"

	"github.com/adamluzsi/frameless"
)

//
// mystorage.go

type MyStorage struct {
	ExternalResourceConnection interface{}
}

func (storage *MyStorage) Close() error {
	closer, ok := storage.ExternalResourceConnection.(io.Closer)

	if !ok {
		return nil
	}

	return closer.Close()
}

func (storage *MyStorage) Store(e frameless.Entity) error {
	switch e.(type) {
	case *MyEntity:
		myEntity := e.(*MyEntity)
		fmt.Println("save in db", myEntity)
		return queries.SetID(myEntity, "42")

	default:
		panic("not implemented")

	}
}

func (storage *MyStorage) Exec(quc frameless.Query) frameless.Iterator {
	switch quc := quc.(type) {
	case queries.FindByID:
		// implementation for queries.DeleteByID with the given external resource connection

		fmt.Printf("searching in %s table for %s ID\n", reflects.FullyQualifiedName(quc.Type), quc.ID)

		return iterators.NewEmpty()
	case queries.DeleteEntity:

		ID, found := queries.LookupID(quc.Entity)

		if !found {
			return iterators.Errorf("this implementation depending on an ID field in the entity")
		}

		name := reflects.FullyQualifiedName(quc.Entity)

		fmt.Printf("deleting from %s the row with the %s ID of\n", name, ID)

		return nil

	default:
		panic("not implemented")

	}
}

//
// mycustomstorage_test.go

func ThisIsHowYouCanCreateTestToTestQueryUseCaseIntegrationsIntoTheStorage(suite *testing.T) {
	suite.Run("Query", func(spec *testing.T) {
		var storage *MyStorage
		reset := func() { *storage = MyStorage{ExternalResourceConnection: nil} }
		reset()
		// or you can create NewMyStorage(interface{}) as well for controlled initialization of your storage implementation,
		// and use it here for initialize the object

		spec.Run("queries.FindByID", func(t *testing.T) {

			// this will test our implementation against the expected behavior in the DeleteByID specification
			queries.FindByID{Type: MyEntity{}}.Test(t, storage)
		})

	})
}

func ExampleStorage() {
	// for working implementation example check frameless/storages package in Memory storage code and test
}
