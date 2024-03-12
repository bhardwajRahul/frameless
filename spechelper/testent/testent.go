package testent

import (
	"context"
	"go.llib.dev/frameless/ports/pubsub"
	"go.llib.dev/testcase"
	"testing"
)

type Foo struct {
	ID  FooID `ext:"ID"`
	Foo string
	Bar string
	Baz string
}

type FooID string

func (f Foo) GetFoo() string {
	return f.Foo
}

func MakeFoo(tb testing.TB) Foo {
	te := testcase.ToT(&tb).Random.Make(Foo{}).(Foo)
	te.ID = ""
	return te
}

func MakeFooFunc(tb testing.TB) func() Foo {
	return func() Foo { return MakeFoo(tb) }
}

type FooDTO struct {
	ID  string `ext:"ID" json:"id"`
	Foo string `json:"foo"`
	Bar string `json:"bar"`
	Baz string `json:"baz"`
}

type FooJSONMapping struct{}

func (n FooJSONMapping) ToDTO(ent Foo) (FooDTO, error) {
	return FooDTO{ID: string(ent.ID), Foo: ent.Foo, Bar: ent.Bar, Baz: ent.Baz}, nil
}

func (n FooJSONMapping) ToEnt(dto FooDTO) (Foo, error) {
	return Foo{ID: FooID(dto.ID), Foo: dto.Foo, Bar: dto.Bar, Baz: dto.Baz}, nil
}

func MakeContextFunc(tb testing.TB) func() context.Context {
	return func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		tb.Cleanup(cancel)
		return ctx
	}
}

type FooQueueID string

type FooQueue struct {
	ID FooQueueID `ext:"id"`
	pubsub.Publisher[Foo]
	pubsub.Subscriber[Foo]
}

func (fq FooQueue) SetPublisher(p pubsub.Publisher[Foo])   { fq.Publisher = p }
func (fq FooQueue) SetSubscriber(s pubsub.Subscriber[Foo]) { fq.Subscriber = s }

type Fooer interface {
	GetFoo() string
}
