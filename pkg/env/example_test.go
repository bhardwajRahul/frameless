package env_test

import (
	"go.llib.dev/frameless/pkg/env"
	"go.llib.dev/frameless/pkg/logger"
	"os"
	"time"
)

func ExampleLoad() {
	type ExampleAppConfig struct {
		Foo string        `env:"FOO"`
		Bar time.Duration `env:"BAR" default:"1h5m"`
		Baz int           `env:"BAZ" required:"true"`
	}

	var c ExampleAppConfig
	if err := env.Load(&c); err != nil {
		logger.Fatal(nil, "failed to load application config", logger.ErrField(err))
		os.Exit(1)
	}
}

func ExampleLoad_enum() {
	type ExampleAppConfig struct {
		Foo string `env:"FOO" enum:"foo;bar;baz;" default:"foo"`
	}

	var c ExampleAppConfig
	if err := env.Load(&c); err != nil {
		logger.Fatal(nil, "failed to load application config", logger.ErrField(err))
		os.Exit(1)
	}
}

func ExampleLookup() {
	val, ok, err := env.Lookup[string]("FOO", env.DefaultValue("foo"))
	_, _, _ = val, ok, err
}

func ExampleLoad_withDefaultValue() {
	type ExampleAppConfig struct {
		Foo string `env:"FOO" default:"foo"`
	}
}

func ExampleLoad_withTimeLayout() {
	type ExampleAppConfig struct {
		Foo time.Time `env:"FOO" layout:"2006-01-02"`
	}
}
