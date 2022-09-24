package containers_test

import (
	"github.com/adamluzsi/frameless/pkg/containers"
	"testing"

	"github.com/adamluzsi/testcase/assert"
	"github.com/adamluzsi/testcase/random"
)

func TestStack(t *testing.T) {
	t.Run("on nil stack", func(t *testing.T) {
		expected := random.New(random.CryptoSeed{}).Int()
		var stack containers.Stack[int]
		_, ok := stack.Last()
		assert.False(t, ok)
		assert.True(t, stack.IsEmpty())
		_, ok = stack.Pop()
		assert.False(t, ok)
		assert.True(t, stack.IsEmpty())
		stack.Push(expected)
		assert.False(t, stack.IsEmpty())
		got, ok := stack.Last()
		assert.True(t, ok)
		assert.Equal(t, expected, got)
		assert.False(t, stack.IsEmpty())
		got, ok = stack.Pop()
		assert.True(t, ok)
		assert.Equal(t, expected, got)
		assert.True(t, stack.IsEmpty())
	})
	t.Run("on empty stack", func(t *testing.T) {
		expected := random.New(random.CryptoSeed{}).Int()
		stack := containers.Stack[int]{}
		_, ok := stack.Last()
		assert.False(t, ok)
		assert.True(t, stack.IsEmpty())
		_, ok = stack.Pop()
		assert.False(t, ok)
		assert.True(t, stack.IsEmpty())
		stack.Push(expected)
		assert.False(t, stack.IsEmpty())
		got, ok := stack.Last()
		assert.True(t, ok)
		assert.Equal(t, expected, got)
		assert.False(t, stack.IsEmpty())
		got, ok = stack.Pop()
		assert.True(t, ok)
		assert.Equal(t, expected, got)
		assert.True(t, stack.IsEmpty())
	})
}