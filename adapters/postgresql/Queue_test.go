package postgresql_test

import (
	"context"
	"github.com/adamluzsi/frameless/adapters/postgresql"
	sh "github.com/adamluzsi/frameless/adapters/postgresql/internal/spechelper"
	"github.com/adamluzsi/frameless/ports/migration"
	"github.com/adamluzsi/frameless/ports/pubsub/pubsubcontracts"
	"github.com/adamluzsi/frameless/ports/pubsub/pubsubtest"
	"github.com/adamluzsi/frameless/spechelper/testent"
	"github.com/adamluzsi/testcase"
	"github.com/adamluzsi/testcase/assert"
	"github.com/adamluzsi/testcase/clock/timecop"
	"github.com/adamluzsi/testcase/pp"
	"github.com/adamluzsi/testcase/random"
	"reflect"
	"testing"
	"time"
)

var _ migration.Migratable = postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{}

func TestQueue(t *testing.T) {
	const queueName = "test_entity"
	c := GetConnection(t)

	assert.NoError(t,
		postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{Name: queueName, Connection: c}.
			Migrate(sh.MakeContext(t)))

	mapping := sh.TestEntityJSONMapping{}

	testcase.RunSuite(t,
		pubsubcontracts.FIFO[sh.TestEntity](func(tb testing.TB) pubsubcontracts.FIFOSubject[sh.TestEntity] {
			q := postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{
				Name:       queueName,
				Connection: c,
				Mapping:    mapping,
			}
			return pubsubcontracts.FIFOSubject[sh.TestEntity]{
				PubSub: pubsubcontracts.PubSub[sh.TestEntity]{
					Publisher:  q,
					Subscriber: q,
				},
				MakeContext: context.Background,
				MakeData:    sh.MakeTestEntityFunc(tb),
			}
		}),
		pubsubcontracts.LIFO[sh.TestEntity](func(tb testing.TB) pubsubcontracts.LIFOSubject[sh.TestEntity] {
			q := postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{
				Name:       queueName,
				Connection: c,
				Mapping:    mapping,

				LIFO: true,
			}
			return pubsubcontracts.LIFOSubject[sh.TestEntity]{
				PubSub: pubsubcontracts.PubSub[sh.TestEntity]{
					Publisher:  q,
					Subscriber: q,
				},
				MakeContext: context.Background,
				MakeData:    sh.MakeTestEntityFunc(tb),
			}
		}),
		pubsubcontracts.Buffered[sh.TestEntity](func(tb testing.TB) pubsubcontracts.BufferedSubject[sh.TestEntity] {
			q := postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{
				Name:       queueName,
				Connection: c,
				Mapping:    mapping,
			}
			return pubsubcontracts.BufferedSubject[sh.TestEntity]{
				PubSub: pubsubcontracts.PubSub[sh.TestEntity]{
					Publisher:  q,
					Subscriber: q,
				},
				MakeContext: context.Background,
				MakeData:    sh.MakeTestEntityFunc(tb),
			}
		}),
		pubsubcontracts.Blocking[sh.TestEntity](func(tb testing.TB) pubsubcontracts.BlockingSubject[sh.TestEntity] {
			q := postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{
				Name:       queueName,
				Connection: c,
				Mapping:    mapping,
		
				Blocking: true,
			}
			return pubsubcontracts.BlockingSubject[sh.TestEntity]{
				PubSub: pubsubcontracts.PubSub[sh.TestEntity]{
					Publisher:  q,
					Subscriber: q,
				},
				MakeContext: context.Background,
				MakeData:    sh.MakeTestEntityFunc(tb),
			}
		}),
		pubsubcontracts.Queue[sh.TestEntity](func(tb testing.TB) pubsubcontracts.QueueSubject[sh.TestEntity] {
			q := postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{
				Name:       queueName,
				Connection: c,
				Mapping:    mapping,
			}
			return pubsubcontracts.QueueSubject[sh.TestEntity]{
				PubSub: pubsubcontracts.PubSub[sh.TestEntity]{
					Publisher:  q,
					Subscriber: q,
				},
				MakeContext: context.Background,
				MakeData:    sh.MakeTestEntityFunc(tb),
			}
		}),
	)
}

func TestQueue_emptyQueueBreakTime(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	const queueName = "TestQueue_emptyQueueBreakTime"
	ctx := context.Background()
	now := time.Now().UTC()
	timecop.Travel(t, now)

	q := postgresql.Queue[testent.Foo, testent.FooDTO]{
		Name:                queueName,
		Connection:          GetConnection(t),
		Mapping:             testent.FooJSONMapping{},
		EmptyQueueBreakTime: time.Hour,
	}
	assert.NoError(t, q.Migrate(sh.MakeContext(t)))

	res := pubsubtest.Subscribe[testent.Foo](t, q, ctx)

	t.Log("we wait until the subscription is idle")
	idler, ok := res.Subscription().(interface{ IsIdle() bool })
	assert.True(t, ok)
	assert.EventuallyWithin(5*time.Second).Assert(t, func(it assert.It) {
		it.Should.True(idler.IsIdle())
	})

	waitTime := 256 * time.Millisecond
	time.Sleep(waitTime)
	
	foo := testent.MakeFoo(t)
	assert.NoError(t, q.Publish(ctx, foo))

	assert.NotWithin(t, waitTime, func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, got := range res.Values() {
					if reflect.DeepEqual(foo, got) {
						return
					}
				}
			}
		}
	})
	
	timecop.Travel(t, time.Hour+time.Second)

	assert.Within(t, waitTime, func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, got := range res.Values() {
					if reflect.DeepEqual(foo, got) {
						return
					}
				}
			}
		}
	})
}

func TestQueue_smoke(t *testing.T) {
	rnd := random.New(random.CryptoSeed{})
	cm := GetConnection(t)
	t.Run("single", func(t *testing.T) {
		q1 := postgresql.Queue[testent.Foo, testent.FooDTO]{
			Name:       "42",
			Connection: cm,
			Mapping:    testent.FooJSONMapping{},
		}

		res1 := pubsubtest.Subscribe[testent.Foo](t, q1, context.Background())

		var (
			ent1A     = rnd.Make(testent.Foo{}).(testent.Foo)
			ent1B     = rnd.Make(testent.Foo{}).(testent.Foo)
			ent1C     = rnd.Make(testent.Foo{}).(testent.Foo)
			expected1 = []testent.Foo{ent1A, ent1B, ent1C}
		)

		assert.NoError(t, q1.Publish(context.Background(), ent1A, ent1B, ent1C))

		res1.Eventually(t, func(tb testing.TB, foos []testent.Foo) {
			assert.ContainExactly(tb, expected1, foos)
		})
	})
	t.Run("multi", func(t *testing.T) {
		cm := GetConnection(t)

		q1 := postgresql.Queue[testent.Foo, testent.FooDTO]{
			Name:       "42",
			Connection: cm,
			Mapping:    testent.FooJSONMapping{},
		}

		q2 := postgresql.Queue[testent.Foo, testent.FooDTO]{
			Name:       "24",
			Connection: cm,
			Mapping:    testent.FooJSONMapping{},
		}

		res1 := pubsubtest.Subscribe[testent.Foo](t, q1, context.Background())
		res2 := pubsubtest.Subscribe[testent.Foo](t, q2, context.Background())

		var (
			rnd = random.New(random.CryptoSeed{})

			ent1A     = rnd.Make(testent.Foo{}).(testent.Foo)
			ent1B     = rnd.Make(testent.Foo{}).(testent.Foo)
			ent1C     = rnd.Make(testent.Foo{}).(testent.Foo)
			expected1 = []testent.Foo{ent1A, ent1B, ent1C}

			ent2A     = rnd.Make(testent.Foo{}).(testent.Foo)
			ent2B     = rnd.Make(testent.Foo{}).(testent.Foo)
			ent2C     = rnd.Make(testent.Foo{}).(testent.Foo)
			expected2 = []testent.Foo{ent2A, ent2B, ent2C}
		)

		assert.NoError(t, q1.Publish(context.Background(), ent1A, ent1B, ent1C))
		assert.NoError(t, q2.Publish(context.Background(), ent2A, ent2B, ent2C))

		t.Cleanup(func() {
			if !t.Failed() {
				return
			}
			t.Log("res1", pp.Format(res1.Values()))
			t.Log("res2", pp.Format(res2.Values()))
		})

		res1.Eventually(t, func(tb testing.TB, foos []testent.Foo) {
			assert.ContainExactly(tb, expected1, foos)
		})

		res2.Eventually(t, func(tb testing.TB, foos []testent.Foo) {
			assert.ContainExactly(tb, expected2, foos)
		})
	})
}

func BenchmarkQueue(b *testing.B) {
	const queueName = "test_entity"
	var (
		ctx = sh.MakeContext(b)
		rnd = random.New(random.CryptoSeed{})
		cm  = GetConnection(b)
		q   = postgresql.Queue[sh.TestEntity, sh.TestEntityDTO]{
			Name:       queueName,
			Connection: cm,
			Mapping:    sh.TestEntityJSONMapping{},
		}
	)

	b.Run("single publish", func(b *testing.B) {
		assert.NoError(b, q.Purge(ctx))
		msgs := random.Slice(b.N, func() sh.TestEntity {
			return sh.TestEntity{
				ID:  rnd.UUID(),
				Foo: rnd.UUID(),
				Bar: rnd.UUID(),
				Baz: rnd.UUID(),
			}
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = q.Publish(ctx, msgs[i])
		}
	})

	b.Run("single element fetch", func(b *testing.B) {
		assert.NoError(b, q.Purge(ctx))
		assert.NoError(b, q.Publish(ctx, random.Slice(b.N, func() sh.TestEntity {
			return sh.TestEntity{
				ID:  rnd.UUID(),
				Foo: rnd.UUID(),
				Bar: rnd.UUID(),
				Baz: rnd.UUID(),
			}
		})...))
		sub := q.Subscribe(ctx)
		defer sub.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if !sub.Next() {
				b.FailNow()
			}
			_ = sub.Value()
		}
	})

	b.Run("batch publish 100", func(b *testing.B) {
		assert.NoError(b, q.Purge(ctx))
		msgs := random.Slice(100, func() sh.TestEntity {
			return sh.TestEntity{
				ID:  rnd.UUID(),
				Foo: rnd.UUID(),
				Bar: rnd.UUID(),
				Baz: rnd.UUID(),
			}
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = q.Publish(ctx, msgs...)
		}
	})
}
