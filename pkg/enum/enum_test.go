package enum_test

import (
	"testing"

	"github.com/adamluzsi/frameless/pkg/enum"
	"github.com/adamluzsi/testcase"
	"github.com/adamluzsi/testcase/assert"
)

func ExampleValidateStruct_string() {
	type ExampleStruct struct {
		V string `enum:"A;B;C;"`
	}

	_ = enum.ValidateStruct(ExampleStruct{V: "A"}) // no error
	_ = enum.ValidateStruct(ExampleStruct{V: "D"}) // has error
}

func ExampleValidateStruct_int() {
	type ExampleStruct struct {
		V int `enum:"2,4,8,16,42,"`
	}

	_ = enum.ValidateStruct(ExampleStruct{V: 42}) // no error
	_ = enum.ValidateStruct(ExampleStruct{V: 24}) // has error
}

func ExampleValidateStruct_float() {
	type ExampleStruct struct {
		V float64 `enum:"2.5;4.2;"`
	}

	_ = enum.ValidateStruct(ExampleStruct{V: 4.2})   // no error
	_ = enum.ValidateStruct(ExampleStruct{V: 24.42}) // has error
}

func ExampleValidateStruct_slice() {
	type ExampleStruct struct {
		V []string `enum:"FOO|BAR|BAZ|"`
	}

	_ = enum.ValidateStruct(ExampleStruct{V: []string{"FOO", "BAR", "BAZ"}}) // no error
	_ = enum.ValidateStruct(ExampleStruct{V: []string{"FOO", "BAB", "BAZ"}}) // has error because of BAB
}

func TestValidateStruct_smoke(t *testing.T) {
	type (
		EmptyEnumExample struct {
			V string `enum:""`
		}
		UnsupportedEnumExample struct {
			V func() `enum:";"`
		}
		InvalidExample struct {
			V int `enum:"hello;world;"`
		}
		StringExample struct {
			V string `enum:"A|B|C|"`
		}
		IntExample struct {
			V int `enum:"42|24|"`
		}
		Int8Example struct {
			V int8 `enum:"42,24,"`
		}
		Int16Example struct {
			V int16 `enum:"42;24;"`
		}
		Int32Example struct {
			V int32 `enum:"42/24/"`
		}
		Int64Example struct {
			V int64 `enum:"42/24/"`
		}
		UIntExample struct {
			V uint `enum:"42|24|"`
		}
		UInt8Example struct {
			V uint8 `enum:"42,24,"`
		}
		UInt16Example struct {
			V uint16 `enum:"42;24;"`
		}
		UInt32Example struct {
			V uint32 `enum:"42/24/"`
		}
		UInt64Example struct {
			V uint64 `enum:"42/24/"`
		}
		Float32Example struct {
			V float32 `enum:"42.24 24.42 "`
		}
		Float64Example struct {
			V float64 `enum:"42.24;24.42;"`
		}
		BoolExample struct {
			V bool `enum:"true;"`
		}
		SubStringType        string
		StringSubTypeExample struct {
			V SubStringType `enum:"A;B;C;"`
		}
		SliceExample struct {
			V []string `enum:"A;B;C;"`
		}
	)

	type Case struct {
		V     any
		IsErr bool
	}
	cases := map[string]Case{
		"on non struct value type, validation fails": {
			V:     "Hello, world!",
			IsErr: true,
		},

		"on empty enum list, everything is accepted": {
			V:     EmptyEnumExample{V: "foo/bar/baz"},
			IsErr: false,
		},

		"on unsupported enum list, error returned": {
			V:     UnsupportedEnumExample{V: func() {}},
			IsErr: true,
		},

		"on invalid enumerator type, error returned": {
			V:     InvalidExample{V: 42},
			IsErr: true,
		},

		"on sub type, when value match the enum list": {
			V:     StringSubTypeExample{V: "A"},
			IsErr: false,
		},
		"on sub type, when value doesn't match the enum list": {
			V:     StringSubTypeExample{V: "foo"},
			IsErr: true,
		},

		"bool - match enum list": {
			V:     BoolExample{V: true},
			IsErr: false,
		},
		"bool - doesn't match enum list": {
			V:     BoolExample{V: false},
			IsErr: true,
		},

		"string - match enum list - enum pos 1": {
			V:     StringExample{V: "A"},
			IsErr: false,
		},
		"string - match enum list - enum pos 2": {
			V:     StringExample{V: "B"},
			IsErr: false,
		},
		"string - doesn't match enum list - invalid value": {
			V:     StringExample{V: "128"},
			IsErr: true,
		},
		"string - doesn't match enum list - zero value, when zero is not registered as valid value": {
			V:     StringExample{},
			IsErr: true,
		},

		"int - match enum list": {
			V:     IntExample{V: 42},
			IsErr: false,
		},
		"int - doesn't match enum list": {
			V:     IntExample{V: 128},
			IsErr: true,
		},

		"int8 - match enum list": {
			V:     Int8Example{V: 42},
			IsErr: false,
		},
		"int8 - doesn't match enum list": {
			V:     Int8Example{V: 16},
			IsErr: true,
		},

		"int16 - match enum list": {
			V:     Int16Example{V: 42},
			IsErr: false,
		},
		"int16 - doesn't match enum list": {
			V:     Int16Example{V: 128},
			IsErr: true,
		},

		"int32 - match enum list": {
			V:     Int32Example{V: 42},
			IsErr: false,
		},
		"int32 - doesn't match enum list": {
			V:     Int32Example{V: 128},
			IsErr: true,
		},

		"int64 - match enum list": {
			V:     Int64Example{V: 42},
			IsErr: false,
		},
		"int64 - doesn't match enum list": {
			V:     Int64Example{V: 128},
			IsErr: true,
		},

		"uint - match enum list": {
			V:     UIntExample{V: 42},
			IsErr: false,
		},
		"uint - doesn't match enum list": {
			V:     UIntExample{V: 128},
			IsErr: true,
		},

		"uint8 - match enum list": {
			V:     UInt8Example{V: 42},
			IsErr: false,
		},
		"uint8 - doesn't match enum list": {
			V:     UInt8Example{V: 16},
			IsErr: true,
		},

		"uint16 - match enum list": {
			V:     UInt16Example{V: 42},
			IsErr: false,
		},
		"uint16 - doesn't match enum list": {
			V:     UInt16Example{V: 128},
			IsErr: true,
		},

		"uint32 - match enum list": {
			V:     UInt32Example{V: 42},
			IsErr: false,
		},
		"uint32 - doesn't match enum list": {
			V:     UInt32Example{V: 128},
			IsErr: true,
		},

		"uint64 - match enum list": {
			V:     UInt64Example{V: 42},
			IsErr: false,
		},
		"uint64 - doesn't match enum list": {
			V:     UInt64Example{V: 128},
			IsErr: true,
		},

		"float32 - match enum list": {
			V:     Float32Example{V: 42.24},
			IsErr: false,
		},
		"float32 - doesn't match enum list": {
			V:     Float32Example{V: 42.42},
			IsErr: true,
		},

		"float64 - match enum list": {
			V:     Float64Example{V: 42.24},
			IsErr: false,
		},
		"float64 - doesn't match enum list": {
			V:     Float64Example{V: 42.42},
			IsErr: true,
		},

		"slice - values match the enum list": {
			V:     SliceExample{V: []string{"A", "C"}},
			IsErr: false,
		},
		"slice - values doesn't match the enum list": {
			V:     SliceExample{V: []string{"A", "foo"}},
			IsErr: true,
		},
	}

	testcase.TableTest(t, cases, func(t *testcase.T, c Case) {
		gotErr := enum.ValidateStruct(c.V)

		if c.IsErr {
			t.Must.Error(gotErr)
		} else {
			t.Must.NoError(gotErr)
		}
	})

	t.Run("test position", func(t *testing.T) {
		assert.Error(t, enum.ValidateStruct(StringExample{}))
		assert.Error(t, enum.ValidateStruct(StringExample{V: "42"}))
		assert.NoError(t, enum.ValidateStruct(StringExample{V: "A"}))
		assert.NoError(t, enum.ValidateStruct(StringExample{V: "B"}))
		assert.NoError(t, enum.ValidateStruct(StringExample{V: "C"}))
	})
}