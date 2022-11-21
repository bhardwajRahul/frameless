package errorutil_test

import (
	"context"
	"errors"
	"github.com/adamluzsi/frameless/pkg/errorutil"
	"github.com/adamluzsi/testcase"
	"github.com/adamluzsi/testcase/let"
	"github.com/adamluzsi/testcase/random"
	"testing"
)

func TestWith_smoke(tt *testing.T) {
	s := testcase.NewSpec(tt)
	var (
		err    = let.Error(s)
		detail = let.String(s)

		ctxKey = let.StringNC(s, 5, random.CharsetAlpha())
		ctxVal = let.String(s)
		ctx    = let.Context(s).Let(s, func(t *testcase.T) context.Context {
			return context.WithValue(context.Background(), ctxKey.Get(t), ctxVal.Get(t))
		})
	)
	t := testcase.NewT(tt, s)

	v := errorutil.With{Err: err.Get(t)}.
		Context(ctx.Get(t)).
		Detail(detail.Get(t))

	t.Must.ErrorIs(err.Get(t), v)
	t.Must.Contain(v.Error(), err.Get(t).Error())
	t.Must.Contain(v.Error(), detail.Get(t))

	gotCtx, ok := errorutil.LookupContext(v)
	t.Must.True(ok)
	t.Must.Equal(ctx.Get(t), gotCtx)
	t.Must.Equal(ctxVal.Get(t), gotCtx.Value(ctxKey.Get(t)).(string))

	gotDetail, ok := errorutil.LookupDetail(v)
	t.Must.True(ok)
	t.Must.Equal(detail.Get(t), gotDetail)
}

func TestWith_Context(t *testing.T) {
	s := testcase.NewSpec(t)

	var (
		err = let.Error(s)
		ctx = let.Context(s).Let(s, func(t *testcase.T) context.Context {
			return context.WithValue(context.WithValue(context.Background(),
				"foo", "bar"),
				"oof", "rab")
		})
	)
	act := func(t *testcase.T) error {
		return errorutil.With{err.Get(t)}.Context(ctx.Get(t))
	}

	s.Then("context can be looked up", func(t *testcase.T) {
		_, ok := errorutil.LookupContext(err.Get(t))
		t.Must.False(ok)

		gotCtx, ok := errorutil.LookupContext(act(t))
		t.Must.True(ok)
		t.Must.Equal(ctx.Get(t), gotCtx)
		t.Must.Equal("bar", gotCtx.Value("foo").(string))
	})

	s.Then(".Error() returns the underlying error's result", func(t *testcase.T) {
		t.Must.Equal(err.Get(t).Error(), act(t).Error())
	})

	s.When("the input error has a typed error", func(s *testcase.Spec) {
		expectedTypedError := testcase.Let(s, func(t *testcase.T) errorutil.UserError {
			return errorutil.UserError{
				ID:      "foo-bar-baz",
				Message: "The foo, bar and the baz",
			}
		})
		err.Let(s, func(t *testcase.T) error {
			return expectedTypedError.Get(t)
		})

		s.Then("the typed error can be looked up with errors.As", func(t *testcase.T) {
			var usrErr errorutil.UserError
			t.Must.True(errors.As(act(t), &usrErr))
			t.Must.Equal(expectedTypedError.Get(t), usrErr)
		})

		s.Then("we can check after the typed error with errors.Is", func(t *testcase.T) {
			t.Must.True(errors.Is(act(t), expectedTypedError.Get(t)))
		})
	})
}

func TestWith_Detail(t *testing.T) {
	s := testcase.NewSpec(t)

	var (
		err    = let.Error(s)
		detail = testcase.Let(s, func(t *testcase.T) string {
			return t.Random.Error().Error()
		})
	)
	act := func(t *testcase.T) error {
		return errorutil.With{err.Get(t)}.Detail(detail.Get(t))
	}

	s.Then("detail can be looked up", func(t *testcase.T) {
		_, ok := errorutil.LookupDetail(err.Get(t))
		t.Must.False(ok)

		gotDetail, ok := errorutil.LookupDetail(act(t))
		t.Must.True(ok)
		t.Must.Equal(detail.Get(t), gotDetail)
	})

	s.Then(".Error() includes the underlying error's result", func(t *testcase.T) {
		t.Must.Contain(act(t).Error(), err.Get(t).Error())
	})

	s.Then(".Error() includes the detail in error's result", func(t *testcase.T) {
		t.Must.Contain(act(t).Error(), detail.Get(t))
	})

	s.When("the input error has a typed error", func(s *testcase.Spec) {
		expectedTypedError := testcase.Let(s, func(t *testcase.T) errorutil.UserError {
			return errorutil.UserError{
				ID:      "foo-bar-baz",
				Message: "The foo, bar and the baz",
			}
		})
		err.Let(s, func(t *testcase.T) error {
			return expectedTypedError.Get(t)
		})

		s.Then("the typed error can be looked up with errors.As", func(t *testcase.T) {
			var usrErr errorutil.UserError
			t.Must.True(errors.As(act(t), &usrErr))
			t.Must.Equal(expectedTypedError.Get(t), usrErr)
		})

		s.Then("we can check after the typed error with errors.Is", func(t *testcase.T) {
			t.Must.True(errors.Is(act(t), expectedTypedError.Get(t)))
		})
	})
}

func TestWith_Detailf(t *testing.T) {
	s := testcase.NewSpec(t)

	var (
		format = testcase.LetValue(s, "%s 42")
		err    = let.Error(s)
		detail = testcase.Let(s, func(t *testcase.T) string {
			return t.Random.Error().Error()
		})
	)
	act := func(t *testcase.T) error {
		return errorutil.With{err.Get(t)}.Detailf(format.Get(t), detail.Get(t))
	}

	s.Then("detail can be looked up", func(t *testcase.T) {
		_, ok := errorutil.LookupDetail(err.Get(t))
		t.Must.False(ok)

		gotDetail, ok := errorutil.LookupDetail(act(t))
		t.Must.True(ok)
		t.Must.Equal(detail.Get(t)+" 42", gotDetail)
	})

	s.Then(".Error() includes the underlying error's result", func(t *testcase.T) {
		t.Must.Contain(act(t).Error(), err.Get(t).Error())
	})

	s.Then(".Error() includes the formatted detail in error's result", func(t *testcase.T) {
		t.Must.Contain(act(t).Error(), detail.Get(t)+" 42")
	})

	s.When("the input error has a typed error", func(s *testcase.Spec) {
		expectedTypedError := testcase.Let(s, func(t *testcase.T) errorutil.UserError {
			return errorutil.UserError{
				ID:      "foo-bar-baz",
				Message: "The foo, bar and the baz",
			}
		})
		err.Let(s, func(t *testcase.T) error {
			return expectedTypedError.Get(t)
		})

		s.Then("the typed error can be looked up with errors.As", func(t *testcase.T) {
			var usrErr errorutil.UserError
			t.Must.True(errors.As(act(t), &usrErr))
			t.Must.Equal(expectedTypedError.Get(t), usrErr)
		})

		s.Then("we can check after the typed error with errors.Is", func(t *testcase.T) {
			t.Must.True(errors.Is(act(t), expectedTypedError.Get(t)))
		})
	})
}
