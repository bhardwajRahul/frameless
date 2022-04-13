package lazyloading_test

import (
	"testing"

	"github.com/adamluzsi/frameless/lazyloading"
	"github.com/adamluzsi/testcase"
	"github.com/adamluzsi/testcase/assert"
	"github.com/adamluzsi/testcase/random"
)

func TestValues(t *testing.T) {
	s := testcase.NewSpec(t)

	loader := testcase.Let(s, func(t *testcase.T) interface{} {
		return &lazyloading.Values{}
	})
	loaderGet := func(t *testcase.T) *lazyloading.Values {
		return loader.Get(t).(*lazyloading.Values)
	}

	s.Describe(`.Get`, func(s *testcase.Spec) {
		key := testcase.Let(s, func(t *testcase.T) interface{} {
			return t.Random.Int()
		})
		initCallCount := s.LetValue(`init call count`, int(0))
		init := testcase.Let(s, func(t *testcase.T) interface{} {
			return func() interface{} { return t.Random.Int() }
		})
		subject := func(t *testcase.T) interface{} {
			return loaderGet(t).Get(key.Get(t), func() interface{} {
				initCallCount.Set(t, initCallCount.Get(t).(int)+1) // ++
				return init.Get(t).(func() interface{})()
			})
		}

		s.Then(`it yield the same result all the time`, func(t *testcase.T) {
			assert.Must(t).Equal(subject(t), subject(t))
		})

		s.Then(`on multiple call, value constructed only once`, func(t *testcase.T) {
			for i := 0; i < 42; i++ {
				subject(t)
			}
			assert.Must(t).Equal(1, initCallCount.Get(t).(int))
		})

		s.When(`when init block returns with nil`, func(s *testcase.Spec) {
			init.Let(s, func(t *testcase.T) interface{} {
				return func() interface{} { return nil }
			})

			s.Then(`on multiple call, value constructed only once`, func(t *testcase.T) {
				for i := 0; i < 42; i++ {
					subject(t)
				}
				assert.Must(t).Equal(1, initCallCount.Get(t).(int))
			})
		})
	})
}

func BenchmarkLazyLoader_Get(b *testing.B) {
	const sampling = 42

	rnd := random.New(random.CryptoSeed{})
	getIndex := func(len, index int) int {
		for {
			if index < len {
				break
			}
			index -= len
		}
		return index
	}

	b.Run(`accessing value from a map`, func(b *testing.B) {
		vs := make(map[string]func() interface{})
		var keys = make([]string, 0)
		for i := 0; i < sampling; i++ {
			key := rnd.String()
			value := rnd.Int()
			vs[key] = func() interface{} { return value }
			keys = append(keys, key)
		}
		keysLen := len(keys)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = vs[keys[getIndex(keysLen, i)]]()
		}
	})

	b.Run(`accessing a value from .Get`, func(b *testing.B) {
		ll := &lazyloading.Values{}
		keys := make([]int, 0)
		for i := 0; i < sampling; i++ {
			keys = append(keys, i)
			ll.Get(i, func() interface{} { return rnd.Int() })
		}
		keysLen := len(keys)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ll.Get(keys[getIndex(keysLen, i)], func() interface{} { panic("boom") })
		}
	})
}
