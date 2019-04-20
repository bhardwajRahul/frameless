package queries

import (
	"github.com/adamluzsi/frameless/resources"
	"testing"
)

func TestUnexportedEntity(t *testing.T, r resources.Resource) {
	t.Run("test query acceptance with unexported entities", func(suite *testing.T) {
		suite.Run("save", func(t *testing.T) {
			Save{Entity: &unexportedEntity{}}.Test(t, r)
		})
		suite.Run("find", func(spec *testing.T) {
			spec.Run("DeleteByID", func(t *testing.T) {
				FindByID{Type: unexportedEntity{}}.Test(t, r)
			})
			spec.Run("FindAll", func(t *testing.T) {
				FindAll{Type: unexportedEntity{}}.Test(t, r)
			})
		})
		suite.Run("update", func(spec *testing.T) {
			spec.Run("UpdateEntity", func(t *testing.T) {
				UpdateEntity{Entity: unexportedEntity{}}.Test(t, r)
			})
		})
		suite.Run("delete", func(spec *testing.T) {
			spec.Run("DeleteByID", func(t *testing.T) {
				DeleteByID{Type: unexportedEntity{}}.Test(t, r)
			})
			spec.Run("DeleteEntity", func(t *testing.T) {
				DeleteEntity{Entity: unexportedEntity{}}.Test(t, r)
			})
		})
	})
}

type unexportedEntity struct {
	ExtID string `ext:"ID"`
	Data  string
}