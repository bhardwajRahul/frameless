package iterators

import (
	"github.com/adamluzsi/frameless"
)

func Collect[T any](i frameless.Iterator[T]) (vs []T, err error) {
	defer func() {
		closeErr := i.Close()
		if err == nil {
			err = closeErr
		}
	}()
	vs = make([]T, 0)
	for i.Next() {
		vs = append(vs, i.Value())
	}
	return vs, i.Err()
}
