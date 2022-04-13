package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/adamluzsi/frameless/cache"
	. "github.com/adamluzsi/frameless/contracts/asserts"
	"github.com/adamluzsi/frameless/extid"
	"github.com/adamluzsi/testcase/assert"

	"github.com/adamluzsi/frameless"
	"github.com/adamluzsi/frameless/contracts"
	"github.com/adamluzsi/frameless/iterators"
	"github.com/adamluzsi/frameless/reflects"
	"github.com/adamluzsi/testcase"
)

var (
	waiter = testcase.Waiter{
		WaitDuration: time.Millisecond,
		WaitTimeout:  time.Second,
	}
	async = testcase.Retry{Strategy: &waiter}
)

type Manager[Ent, ID any] struct {
	Subject func(testing.TB) ManagerSubject[Ent, ID]
	MakeCtx func(testing.TB) context.Context
	MakeEnt func(testing.TB) Ent
}

type ManagerSubject[Ent, ID any] struct {
	Cache         Cache[Ent, ID]
	Source        cache.Source[Ent, ID]
	CommitManager frameless.OnePhaseCommitProtocol
}

type Cache[Ent, ID any] interface {
	frameless.Creator[Ent]
	frameless.Finder[Ent, ID]
	frameless.Updater[Ent]
	frameless.Deleter[ID]
	frameless.CreatorPublisher[Ent]
	frameless.UpdaterPublisher[Ent]
	frameless.DeleterPublisher[ID]
}

func (c Manager[Ent, ID]) Test(t *testing.T) {
	c.Spec(testcase.NewSpec(t))
}

func (c Manager[Ent, ID]) Benchmark(b *testing.B) {
	c.Spec(testcase.NewSpec(b))
}

func (c Manager[Ent, ID]) ManagerSubject() testcase.Var[ManagerSubject[Ent, ID]] {
	return testcase.Var[ManagerSubject[Ent, ID]]{
		ID:   "ManagerSubject[Ent, ID]",
		Init: func(t *testcase.T) ManagerSubject[Ent, ID] { return c.Subject(t) },
	}
}

func (c Manager[Ent, ID]) Spec(s *testcase.Spec) {
	newManager := func(tb testing.TB) Cache[Ent, ID] {
		return c.ManagerSubject().Get(tb.(*testcase.T)).Cache
	}

	testcase.RunContract(s,
		contracts.Creator[Ent, ID]{
			Subject: func(tb testing.TB) contracts.CreatorSubject[Ent, ID] {
				return newManager(tb)
			},
			MakeEnt: c.MakeEnt,
			MakeCtx: c.MakeCtx,
		},
		contracts.Finder[Ent, ID]{
			Subject: func(tb testing.TB) contracts.FinderSubject[Ent, ID] {
				return newManager(tb)
			},
			MakeEnt: c.MakeEnt,
			MakeCtx: c.MakeCtx,
		},
		contracts.Deleter[Ent, ID]{
			Subject: func(tb testing.TB) contracts.DeleterSubject[Ent, ID] {
				return newManager(tb)
			},
			MakeEnt: c.MakeEnt,
			MakeCtx: c.MakeCtx,
		},
		contracts.Publisher[Ent, ID]{
			Subject: func(tb testing.TB) contracts.PublisherSubject[Ent, ID] {
				ms := c.Subject(tb)
				if _, ok := ms.Source.(frameless.Updater[Ent]); !ok {
					tb.Skip()
				}
				return ms.Cache
			},
			MakeEnt: c.MakeEnt,
			MakeCtx: c.MakeCtx,
		},
		contracts.Updater[Ent, ID]{
			Subject: func(tb testing.TB) contracts.UpdaterSubject[Ent, ID] {
				ms := c.Subject(tb)
				if _, ok := ms.Source.(frameless.Updater[Ent]); !ok {
					tb.Skip()
				}
				return ms.Cache
			},
			MakeEnt: c.MakeEnt,
			MakeCtx: c.MakeCtx,
		},
		contracts.OnePhaseCommitProtocol[Ent, ID]{
			Subject: func(tb testing.TB) contracts.OnePhaseCommitProtocolSubject[Ent, ID] {
				subject := c.Subject(tb)
				return contracts.OnePhaseCommitProtocolSubject[Ent, ID]{
					Resource:      subject.Cache,
					CommitManager: subject.CommitManager,
				}
			},
			MakeEnt: c.MakeEnt,
			MakeCtx: c.MakeCtx,
		},
	)

	s.Context(``, func(s *testcase.Spec) {
		c.describeResultCaching(s)
		c.describeCacheInvalidationByEventsThatMutatesAnEntity(s)
	})
}

func (c Manager[Ent, ID]) describeCacheInvalidationByEventsThatMutatesAnEntity(s *testcase.Spec) {
	s.Context(reflects.SymbolicName(*new(Ent)), func(s *testcase.Spec) {
		value := testcase.Let(s, func(t *testcase.T) interface{} {
			ptr := c.createT(t)
			assert.Must(t).Nil(c.source().Get(t).Create(c.MakeCtx(t), ptr))
			id, _ := extid.Lookup[ID](ptr)
			t.Defer(c.source().Get(t).DeleteByID, c.MakeCtx(t), id)
			return ptr
		})

		s.Test(`an update to the storage should invalidate the local cache unit entity state`, func(t *testcase.T) {
			v := value.Get(t)
			id, _ := extid.Lookup[ID](v)

			// cache
			_, _ = c.manager().Get(t).FindByID(c.MakeCtx(t), new(Ent), id)   // should trigger caching
			_, _ = iterators.Count(c.manager().Get(t).FindAll(c.MakeCtx(t))) // should trigger caching

			// mutate
			vUpdated := c.createT(t)
			assert.Must(t).Nil(extid.Set(vUpdated, id))
			Update[Ent, ID](t, c.manager().Get(t), c.MakeCtx(t), vUpdated)
			waiter.Wait()

			ptr := IsFindable[Ent, ID](t, c.manager().Get(t), c.MakeCtx(t), id) // should trigger caching
			assert.Must(t).Equal(vUpdated, ptr)
		})

		s.Test(`a delete by id to the storage should invalidate the local cache unit entity state`, func(t *testcase.T) {
			v := value.Get(t)
			id, _ := extid.Lookup[ID](v)

			// cache
			_, _ = c.manager().Get(t).FindByID(c.MakeCtx(t), new(Ent), id)   // should trigger caching
			_, _ = iterators.Count(c.manager().Get(t).FindAll(c.MakeCtx(t))) // should trigger caching

			// delete
			assert.Must(t).Nil(c.manager().Get(t).DeleteByID(c.MakeCtx(t), id))

			// assert
			IsAbsent[Ent, ID](t, c.manager().Get(t), c.MakeCtx(t), id)
		})

		s.Test(`a delete all entity in the storage should invalidate the local cache unit entity state`, func(t *testcase.T) {
			v := value.Get(t)
			id, _ := extid.Lookup[ID](v)

			// cache
			_, _ = c.manager().Get(t).FindByID(c.MakeCtx(t), new(Ent), id)   // should trigger caching
			_, _ = iterators.Count(c.manager().Get(t).FindAll(c.MakeCtx(t))) // should trigger caching

			// delete
			assert.Must(t).Nil(c.manager().Get(t).DeleteAll(c.MakeCtx(t)))
			waiter.Wait()

			IsAbsent[Ent, ID](t, c.manager().Get(t), c.MakeCtx(t), id) // should trigger caching for not found
		})
	})
}

func (c Manager[Ent, ID]) manager() testcase.Var[Cache[Ent, ID]] {
	return testcase.Var[Cache[Ent, ID]]{
		ID: `cache`,
		Init: func(t *testcase.T) Cache[Ent, ID] {
			return c.ManagerSubject().Get(t).Cache
		},
	}
}

func (c Manager[Ent, ID]) source() testcase.Var[cache.Source[Ent, ID]] {
	// source resource where the cache manager retrieve the data in case cache hit is missing
	return testcase.Var[cache.Source[Ent, ID]]{
		ID: `cache manager's source of truth`,
		Init: func(t *testcase.T) cache.Source[Ent, ID] {
			return c.ManagerSubject().Get(t).Source
		},
	}
}

type SpySource[Ent, ID any] struct {
	cache.Source[Ent, ID]
	count struct {
		FindByID int
	}
}

func (stub *SpySource[Ent, ID]) FindByID(ctx context.Context, ptr *Ent, id ID) (_found bool, _err error) {
	stub.count.FindByID++
	return stub.Source.FindByID(ctx, ptr, id)
}

func (c Manager[Ent, ID]) describeResultCaching(s *testcase.Spec) {
	s.Context(reflects.SymbolicName(*new(Ent)), func(s *testcase.Spec) {
		value := testcase.Let(s, func(t *testcase.T) interface{} {
			ptr := c.createT(t)
			storage := c.source().Get(t)
			assert.Must(t).Nil(storage.Create(c.MakeCtx(t), ptr))
			id, _ := extid.Lookup[ID](ptr)
			t.Defer(storage.DeleteByID, c.MakeCtx(t), id)
			return ptr
		})

		s.Then(`it will return the value`, func(t *testcase.T) {
			v := new(Ent)
			id, found := extid.Lookup[ID](value.Get(t))
			assert.Must(t).True(found)
			found, err := c.manager().Get(t).FindByID(c.MakeCtx(t), v, id)
			assert.Must(t).Nil(err)
			assert.Must(t).True(found)
			assert.Must(t).Equal(value.Get(t), v)
		})

		s.And(`after value already cached`, func(s *testcase.Spec) {
			s.Before(func(t *testcase.T) {
				id, found := extid.Lookup[ID](value.Get(t))
				assert.Must(t).True(found)
				v := IsFindable[Ent, ID](t, c.source().Get(t), c.MakeCtx(t), id)
				assert.Must(t).Equal(value.Get(t), v)
			})

			s.And(`value is suddenly updated `, func(s *testcase.Spec) {
				valueWithNewContent := testcase.Let(s, func(t *testcase.T) *Ent {
					id, found := extid.Lookup[ID](value.Get(t))
					assert.Must(t).True(found)
					nv := c.createT(t)
					assert.Must(t).Nil(extid.Set(nv, id))
					return nv
				})

				s.Before(func(t *testcase.T) {
					ptr := valueWithNewContent.Get(t)
					Update[Ent, ID](t, c.manager().Get(t), c.MakeCtx(t), ptr)
					waiter.Wait()
				})

				s.Then(`it will return the new value instead the old one`, func(t *testcase.T) {
					id, found := extid.Lookup[ID](value.Get(t))
					assert.Must(t).True(found)
					t.Must.NotEmpty(t, id)
					HasEntity[Ent, ID](t, c.manager().Get(t), c.MakeCtx(t), valueWithNewContent.Get(t))

					async.Assert(t, func(it assert.It) {
						v := new(Ent)
						found, err := c.manager().Get(t).FindByID(c.MakeCtx(t), v, id)
						it.Must.Nil(err)
						it.Must.True(found)
						it.Log(`actually`, v)
						it.Must.Equal(valueWithNewContent.Get(t), v)
					})
				})
			})
		})

		s.And(`on multiple request`, func(s *testcase.Spec) {
			s.Then(`it will return it consistently`, func(t *testcase.T) {
				value := value.Get(t)
				id, found := extid.Lookup[ID](value)
				assert.Must(t).True(found)

				for i := 0; i < 42; i++ {
					v := new(Ent)
					found, err := c.manager().Get(t).FindByID(c.MakeCtx(t), v, id)
					assert.Must(t).Nil(err)
					assert.Must(t).True(found)
					assert.Must(t).Equal(value, v)
				}
			})

			s.When(`the storage is sensitive to continuous requests`, func(s *testcase.Spec) {
				spy := testcase.Let(s, func(t *testcase.T) *SpySource[Ent, ID] {
					return &SpySource[Ent, ID]{Source: c.source().Get(t)}
				})
				s.Before(func(t *testcase.T) {
					c.source().Set(t, spy.Get(t))
				})

				s.Then(`it will only bother the storage for the value once`, func(t *testcase.T) {
					var (
						nv  *Ent
						err error
					)
					value := value.Get(t)
					id, found := extid.Lookup[ID](value)
					assert.Must(t).True(found)

					// trigger caching
					nv = IsFindable[Ent, ID](t, c.manager().Get(t), c.MakeCtx(t), id)
					assert.Must(t).Equal(value, nv)
					numberOfFindByIDCallAfterEntityIsFound := spy.Get(t).count.FindByID
					waiter.Wait()

					nv = new(Ent)
					found, err = c.manager().Get(t).FindByID(c.MakeCtx(t), nv, id) // should use cached value
					assert.Must(t).Nil(err)
					assert.Must(t).True(found)
					assert.Must(t).Equal(value, nv)
					assert.Must(t).Equal(numberOfFindByIDCallAfterEntityIsFound, spy.Get(t).count.FindByID)
				})
			})
		})
	}, testcase.Flaky(time.Minute))
}

func (c Manager[Ent, ID]) createT(t *testcase.T) *Ent {
	ent := c.MakeEnt(t)
	return &ent
}
