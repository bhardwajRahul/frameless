package iterators_test

import (
	"errors"
	"github.com/adamluzsi/testcase/pp"
	"testing"

	"go.llib.dev/frameless/ports/iterators"

	"github.com/adamluzsi/testcase/assert"
)

var _ iterators.Iterator[any] = iterators.Error[any](errors.New("boom"))

func TestNewError_ErrorGiven_NotIterableIteratorReturnedWithError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("Boom!")
	i := iterators.Error[any](expectedError)
	assert.Must(t).False(i.Next())
	assert.Must(t).Nil(i.Value())
	assert.Must(t).NotNil(expectedError, assert.Message(pp.Format(i.Err())))
	assert.Must(t).Nil(i.Close())
}
