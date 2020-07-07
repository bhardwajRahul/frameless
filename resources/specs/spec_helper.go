package specs

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/adamluzsi/frameless/resources"

	"github.com/adamluzsi/testcase"
	"github.com/stretchr/testify/require"

	"github.com/adamluzsi/frameless"
	"github.com/adamluzsi/frameless/reflects"
)

const ErrIDRequired frameless.Error = `
Can't find the ID in the current structure
if there is no ID in the subject structure
custom test needed that explicitly defines how ID is stored and retried from an entity
`

type minimumRequirements interface {
	resources.Creator
	resources.Finder
	resources.Deleter
}

func thenExternalIDFieldIsExpected(s *testcase.Spec, entityType interface{}) {
	entityTypeName := reflects.FullyQualifiedName(entityType)
	desc := fmt.Sprintf(`An ext:ID field is given in %s`, entityTypeName)
	s.Test(desc, func(t *testcase.T) {
		_, hasExtID := resources.LookupID(newEntityBasedOn(entityType))
		require.True(t, hasExtID, ErrIDRequired.Error())
	})
}

func createEntities(f FixtureFactory, T interface{}) []interface{} {
	var es []interface{}
	for i := 0; i < benchmarkEntityVolumeCount; i++ {
		es = append(es, f.Create(T))
	}
	return es
}

func saveEntities(tb testing.TB, s resources.Creator, f FixtureFactory, es ...interface{}) []string {
	var ids []string
	for _, e := range es {
		require.Nil(tb, s.Create(f.Context(), e))
		id, _ := resources.LookupID(e)
		ids = append(ids, id)
	}
	return ids
}

func cleanup(tb testing.TB, t resources.Deleter, f FixtureFactory, T interface{}) {
	require.Nil(tb, t.DeleteAll(f.Context(), T))
}

func contains(tb testing.TB, slice interface{}, contains interface{}, msgAndArgs ...interface{}) {
	containsRefVal := reflect.ValueOf(contains)
	if containsRefVal.Kind() == reflect.Ptr {
		contains = containsRefVal.Elem().Interface()
	}
	require.Contains(tb, slice, contains, msgAndArgs...)
}

func newEntityBasedOn(T interface{}) interface{} {
	return reflect.New(reflect.TypeOf(T)).Interface()
}
