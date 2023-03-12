package pubsubcontracts

import (
	"context"
	"github.com/adamluzsi/frameless/ports/pubsub/pubsubtest"
	"github.com/adamluzsi/testcase"
	"github.com/adamluzsi/testcase/assert"
	"sync/atomic"
	"testing"
	"time"
)

type Blocking[Data any] struct {
	MakeSubject func(testing.TB) PubSub[Data]
	MakeContext func(testing.TB) context.Context
	MakeData    func(testing.TB) Data

	RollbackOnPublishCancellation bool
}

func (c Blocking[Data]) Spec(s *testcase.Spec) {
	b := base[Data]{
		MakeSubject: c.MakeSubject,
		MakeContext: c.MakeContext,
		MakeData:    c.MakeData,
	}
	b.Spec(s)

	s.Context("blocking pub/sub", func(s *testcase.Spec) {
		b.TryCleanup(s)

		sub := b.GivenWeHaveSubscription(s)

		s.Test("publish will block until a subscriber acknowledged the published message", func(t *testcase.T) {
			var publishedAtUNIXMilli int64
			go func() {
				t.Must.NoError(b.subject().Get(t).Publish(c.MakeContext(t), c.MakeData(t)))
				publishedAt := time.Now().UTC()
				atomic.AddInt64(&publishedAtUNIXMilli, publishedAt.UnixMilli())
			}()

			var (
				receivedAt time.Time
				ackedAt    time.Time
			)
			t.Eventually(func(it assert.It) {
				ackedAt = sub.Get(t).AckedAt()
				it.Must.False(ackedAt.IsZero())
				receivedAt = sub.Get(t).ReceivedAt()
				it.Must.False(receivedAt.IsZero())
			})

			var publishedAt time.Time
			t.Eventually(func(t assert.It) {
				unixMilli := atomic.LoadInt64(&publishedAtUNIXMilli)
				t.Must.NotEmpty(unixMilli)
				publishedAt = time.UnixMilli(unixMilli).UTC()
			})

			t.Must.True(receivedAt.Before(publishedAt),
				"it was expected that the message was received before the publish was done")

			t.Must.True(ackedAt.Before(publishedAt),
				"it was expected that acknowledging time is before the publishing time")
		})

		if c.RollbackOnPublishCancellation {
			s.Test("on context cancellation, message publishing is revoked", func(t *testcase.T) {
				sub.Get(t).Stop() // stop processing from avoiding flaky test runs

				ctx, cancel := context.WithCancel(c.MakeContext(t))
				go func() {
					t.Random.Repeat(10, 100, pubsubtest.Waiter.Wait)
					cancel()
				}()

				t.Must.ErrorIs(ctx.Err(), b.subject().Get(t).Publish(ctx, c.MakeData(t)))

				sub.Get(t).Start(t, c.MakeContext(t))

				pubsubtest.Waiter.Wait()

				t.Must.True(sub.Get(t).AckedAt().IsZero())
			})
		}
	})
}

func (c Blocking[Data]) Test(t *testing.T) { c.Spec(testcase.NewSpec(t)) }

func (c Blocking[Data]) Benchmark(b *testing.B) { c.Spec(testcase.NewSpec(b)) }
