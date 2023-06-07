package runtimekit_test

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/adamluzsi/frameless/pkg/runtimekit"
	"github.com/adamluzsi/testcase"
	"github.com/adamluzsi/testcase/assert"
)

func TestRecover(t *testing.T) {
	s := testcase.NewSpec(t)

	var (
		action    = testcase.Var[func() error]{ID: `action`}
		actionLet = func(s *testcase.Spec, fn func() error) { action.Let(s, func(t *testcase.T) func() error { return fn }) }
	)
	subject := func(t *testcase.T) (err error) {
		defer runtimekit.Recover(&err)
		return action.Get(t)()
	}

	s.When(`action ends without error`, func(s *testcase.Spec) {
		actionLet(s, func() error { return nil })

		s.Then(`it will do nothing`, func(t *testcase.T) {
			assert.Must(t).Nil(subject(t))
		})
	})

	s.When(`action returns an error`, func(s *testcase.Spec) {
		expectedErr := errors.New(`boom`)
		actionLet(s, func() error { return expectedErr })

		s.Then(`it will pass the received error through`, func(t *testcase.T) {
			assert.Must(t).Equal(expectedErr, subject(t))
		})
	})

	s.When(`action panics with an error`, func(s *testcase.Spec) {
		expectedErr := errors.New(`boom`)
		actionLet(s, func() error { panic(expectedErr) })

		s.Then(`it will capture the error from panic and returns with it`, func(t *testcase.T) {
			assert.Must(t).Equal(expectedErr, subject(t))
		})
	})

	s.When(`action panics with an error`, func(s *testcase.Spec) {
		expectedErr := errors.New(`boom`)
		actionLet(s, func() error { panic(expectedErr) })

		s.Then(`it will capture the error from panic and returns with it`, func(t *testcase.T) {
			assert.Must(t).Equal(expectedErr, subject(t))
		})
	})

	s.When(`action panics with an non error type`, func(s *testcase.Spec) {
		const msg = `boom`
		actionLet(s, func() error { panic(msg) })

		s.Then(`it will capture the panic value and create an error from it, where message is the panic object is formatted with fmt`, func(t *testcase.T) {
			assert.Must(t).Equal(errors.New("boom"), subject(t))
		})
	})

	s.When(`action stops the go routine`, func(s *testcase.Spec) {
		actionLet(s, func() error {
			runtime.Goexit()
			return nil
		})

		s.Then(`it will let go exit continues`, func(t *testcase.T) {
			var (
				wg       = &sync.WaitGroup{}
				finished bool
			)
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = subject(t)
				finished = true
			}()
			wg.Wait()
			assert.Must(t).False(finished)
		})
	})
}

func TestOnRecover(t *testing.T) {
	s := testcase.NewSpec(t)

	var (
		action    = testcase.Var[func() error]{ID: `action`}
		actionLet = func(s *testcase.Spec, fn func() error) { action.Let(s, func(t *testcase.T) func() error { return fn }) }
	)
	subject := func(t *testcase.T) (err error) {
		defer runtimekit.OnRecover(func(r any) { err = fmt.Errorf("%v", r) })
		return action.Get(t)()
	}

	s.When(`action ends without error`, func(s *testcase.Spec) {
		actionLet(s, func() error { return nil })

		s.Then(`it will do nothing`, func(t *testcase.T) {
			assert.Must(t).Nil(subject(t))
		})
	})

	s.When(`action panics`, func(s *testcase.Spec) {
		const msg = `boom`
		actionLet(s, func() error { panic(msg) })

		s.Then(`it will capture the panic value and create an error from it, where message is the panic object is formatted with fmt`, func(t *testcase.T) {
			assert.Must(t).Equal(errors.New("boom"), subject(t))
		})
	})

	s.When(`action stops the go routine`, func(s *testcase.Spec) {
		actionLet(s, func() error {
			runtime.Goexit()
			return nil
		})

		s.Then(`it will let go exit continues`, func(t *testcase.T) {
			var (
				wg       = &sync.WaitGroup{}
				finished bool
			)
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = subject(t)
				finished = true
			}()
			wg.Wait()
			assert.Must(t).False(finished)
		})
	})
}

func BenchmarkOnRecover(b *testing.B) {
	b.Run("no panic + lambda", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			func() {
				fn := func() {}
				defer func() {
					if r := recover(); r != nil {
						fn()
					}
				}()
				// do something
			}()
		}
	})

	b.Run("no panic + lambda", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			func() {
				fn := func() {}
				defer func() {
					if r := recover(); r != nil {
						fn()
					}
				}()

				panic("boom")
			}()
		}
	})

	b.Run("no panic + OnRecover", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			func() {
				defer runtimekit.OnRecover(func(any) {})
				// do something
			}()
		}
	})
	
	b.Run("panic + OnRecover", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			func() {
				defer runtimekit.OnRecover(func(any) {})
				panic("boom")
			}()
		}
	})
}