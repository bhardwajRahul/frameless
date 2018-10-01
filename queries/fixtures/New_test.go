package fixtures_test

import (
	"github.com/adamluzsi/frameless/queries/fixtures"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("fixtures.New", func(t *testing.T) {
		t.Run("when given value is a struct", func(t *testing.T) {
			SharedSpecAssertions(t, func() *Example {
				return fixtures.New(Example{}).(*Example)
			})
		})

		t.Run("when given value is a pointer to a struct", func(t *testing.T) {
			SharedSpecAssertions(t, func() *Example {
				return fixtures.New(&Example{}).(*Example)
			})
		})
	})
}

func SharedSpecAssertions(t *testing.T, subject func() *Example) {
	require.NotNil(t, subject())

	t.Run("bool", func(t *testing.T) {
		t.Parallel()

		bools := make(map[bool]struct{})

		for i := 0; i < 1024; i++ {
			bools[subject().Bool] = struct{}{}
		}

		if _, ok := bools[true]; !ok {
			t.Fail()
		}

		if _, ok := bools[false]; !ok {
			t.Fail()
		}

	})
	t.Run("string", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, subject().String)
	})
	t.Run("Integer", func(t *testing.T) {
		t.Parallel()

		require.NotEqual(t, subject().Int, 0)
		require.NotEqual(t, subject().Int8, 0)
		require.NotEqual(t, subject().Int16, 0)
		require.NotEqual(t, subject().Int32, 0)
		require.NotEqual(t, subject().Int64, 0)
	})
	t.Run("unsigned Integer", func(t *testing.T) {
		t.Parallel()

		require.NotEqual(t, subject().UInt, 0)
		require.NotEqual(t, subject().UInt8, 0)
		require.NotEqual(t, subject().UInt16, 0)
		require.NotEqual(t, subject().UInt32, 0)
		require.NotEqual(t, subject().UInt64, 0)
	})
	t.Run("uintptr", func(t *testing.T) {
		t.Parallel()

		require.NotEqual(t, subject().UIntPtr, 0)
	})
	t.Run("floating point number", func(t *testing.T) {
		t.Parallel()

		require.NotEqual(t, subject().Float32, 0)
		require.NotEqual(t, subject().Float64, 0)
	})
	t.Run("complex numbers", func(t *testing.T) {
		t.Parallel()

		require.NotEqual(t, subject().Complex64, 0)
		require.NotEqual(t, subject().Complex128, 0)
	})
	t.Run("array", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, subject().ArrayOfInt)
		require.NotNil(t, subject().ArrayOfString)
	})
	t.Run("slice", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, subject().SliceOfInt)
		require.NotNil(t, subject().SliceOfString)
	})
	t.Run("chan", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, subject().ChanOfInt)
		require.NotNil(t, subject().ChanOfString)
	})
	t.Run("map", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, subject().Map)
	})
	t.Run("pointer", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, *subject().StringPtr)
		require.NotNil(t, *subject().IntPtr)
	})
	t.Run("struct", func(t *testing.T) {
		t.Parallel()

		require.NotEqual(t, subject().ExampleStruct.Int, 0)
		require.NotEmpty(t, subject().ExampleStruct.String)
	})
	t.Run("func", func(t *testing.T) {
		t.Parallel()

		require.Nil(t, subject().Func)
	})
}

type Example struct {
	Bool          bool
	String        string
	Int           int
	Int8          int8
	Int16         int16
	Int32         int32
	Int64         int64
	UIntPtr       uintptr
	UInt          uint
	UInt8         uint8
	UInt16        uint16
	UInt32        uint32
	UInt64        uint64
	Float32       float32
	Float64       float64
	Complex64     complex64
	Complex128    complex128
	ArrayOfString [1]string
	ArrayOfInt    [1]int
	SliceOfString []string
	SliceOfInt    []int
	ChanOfString  chan string
	ChanOfInt     chan int
	Map           map[string]int
	StringPtr     *string
	IntPtr        *int
	Func          func()
	ExampleStruct
}

type ExampleStruct struct {
	String string
	Int    int
}
