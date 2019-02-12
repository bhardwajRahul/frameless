package resourcespecs

import (
	"testing"

	"github.com/adamluzsi/frameless"
)

type minimumRequirementsDependency interface {
	Save
	FindByID
	DeleteByID
}

func TestMinimumRequirements(t *testing.T, r minimumRequirementsDependency) {
	t.Run("TestMinimumRequirements", func(t *testing.T) {

		shared := func(t *testing.T, entity frameless.Entity) {
			SaveSpec{Entity: entity, Subject: r}.Test(t)
			FindByIDSpec{Type: entity, Subject: r}.Test(t)
			DeleteByIDSpec{Type: entity, Subject: r}.Test(t)
		}

		shared(t, ExportedEntity{})
		shared(t, unexportedEntity{})

	})
}
