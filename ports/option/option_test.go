package option_test

import (
	"testing"

	"go.llib.dev/frameless/ports/option"
	"go.llib.dev/testcase/assert"
)

var _ option.Option[any] = option.Func[any](nil)

type SampleConfig struct {
	Foo string
	Bar int
	Baz float64
}

func (c *SampleConfig) Init() {
	c.Foo = "foo"
	c.Bar = 42
	c.Baz = 42.24
}

func FooTo(v string) option.Option[SampleConfig] {
	return option.Func[SampleConfig](func(c *SampleConfig) {
		c.Foo = v
	})
}

func BazTo(v float64) option.Option[SampleConfig] {
	return option.Func[SampleConfig](func(c *SampleConfig) {
		c.Baz = v
	})
}

func BarTo(v int) option.Option[SampleConfig] {
	return option.Func[SampleConfig](func(c *SampleConfig) {
		c.Bar = v
	})
}

func TestUse(t *testing.T) {
	t.Run("smoke", func(t *testing.T) {
		opts := []option.Option[SampleConfig]{
			BazTo(128.821),
			BarTo(128),
		}
		c := option.Use[SampleConfig](opts)
		assert.Equal(t, c.Foo, "foo", "value taken from Init")
		assert.Equal(t, c.Bar, 128, "bar option used")
		assert.Equal(t, c.Baz, 128.821, "baz option used")
	})
	t.Run("init", func(t *testing.T) {
		var exp SampleConfig
		exp.Init()
		got := option.Use[SampleConfig]([]option.Option[SampleConfig](nil))
		assert.Equal(t, exp, got)
	})
	t.Run("options used", func(t *testing.T) {
		opts := []option.Option[SampleConfig]{FooTo("OOF")}
		c := option.Use[SampleConfig](opts)
		assert.Equal(t, c.Foo, "OOF")
	})
	t.Run("nil option values are ignored", func(t *testing.T) {
		opts := []option.Option[SampleConfig]{nil, FooTo("OOF")}
		c := option.Use[SampleConfig](opts)
		assert.Equal(t, c.Foo, "OOF")
	})
}
