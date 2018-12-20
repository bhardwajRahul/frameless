package queries_test

import (
	"github.com/adamluzsi/frameless"
	"testing"
)

type testable func(t *testing.T, resource frameless.Resource)

func (fn testable) Test(t *testing.T, resource frameless.Resource) {
	fn(t, resource)
}