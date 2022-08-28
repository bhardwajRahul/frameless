package postgresql_test

import (
	"context"
	"database/sql"
	"github.com/adamluzsi/frameless/ports/comproto/contracts"
	"github.com/adamluzsi/frameless/ports/crud/contracts"
	"github.com/adamluzsi/frameless/ports/meta/contracts"
	pubsubcontracts "github.com/adamluzsi/frameless/ports/pubsub/contracts"
	contracts2 "github.com/adamluzsi/frameless/spechelper/resource"
	"testing"

	"github.com/adamluzsi/frameless/adapters/postgresql"
	"github.com/adamluzsi/testcase/assert"

	psh "github.com/adamluzsi/frameless/adapters/postgresql/spechelper"
	"github.com/stretchr/testify/require"

	"github.com/adamluzsi/testcase"
)

func TestPostgresConnection(t *testing.T) {
	it := assert.MakeIt(t)

	t.Log(psh.DatabaseURL(t))

	db, err := sql.Open("postgres", psh.DatabaseURL(t))
	it.Must.Nil(err)
	it.Must.Nil(db.Ping())

	var b bool
	it.Must.Nil(db.QueryRow("SELECT TRUE").Scan(&b))
	it.Must.True(b)
}

func TestNewStorage_smoke(t *testing.T) {
	storage := NewTestEntityStorage(t)

	ctx := context.Background()

	ent := &psh.TestEntity{
		Foo: "foo",
		Bar: "bar",
		Baz: "baz",
	}

	require.NoError(t, storage.Create(ctx, ent))
	require.NotEmpty(t, ent.ID)

	ent2, found, err := storage.FindByID(ctx, ent.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, *ent, ent2)

	require.NoError(t, storage.DeleteByID(ctx, ent.ID))
	_, found, err = storage.FindByID(ctx, ent.ID)
	require.NoError(t, err)
	require.False(t, found, `should be deleted`)
}

func TestStorage(t *testing.T) {
	mapping := psh.TestEntityMapping()
	cm := postgresql.NewConnectionManager(psh.DatabaseURL(t))
	sm := postgresql.NewListenNotifySubscriptionManager[psh.TestEntity, string](mapping, psh.DatabaseURL(t), cm)
	subject := &postgresql.Storage[psh.TestEntity, string]{
		ConnectionManager:   cm,
		SubscriptionManager: sm,
		Mapping:             mapping,
	}

	psh.MigrateTestEntity(t, cm)

	testcase.RunSuite(t,
		crudcontracts.Creator[psh.TestEntity, string]{
			Subject: func(tb testing.TB) crudcontracts.CreatorSubject[psh.TestEntity, string] { return subject },
			MakeEnt: psh.MakeTestEntity,
			MakeCtx: psh.MakeCtx,
		},
		crudcontracts.Finder[psh.TestEntity, string]{Subject: func(tb testing.TB) crudcontracts.FinderSubject[psh.TestEntity, string] { return subject },
			MakeEnt: psh.MakeTestEntity,
			MakeCtx: psh.MakeCtx,
		},
		crudcontracts.Updater[psh.TestEntity, string]{Subject: func(tb testing.TB) crudcontracts.UpdaterSubject[psh.TestEntity, string] { return subject },
			MakeEnt: psh.MakeTestEntity,
			MakeCtx: psh.MakeCtx,
		},
		crudcontracts.Deleter[psh.TestEntity, string]{Subject: func(tb testing.TB) crudcontracts.DeleterSubject[psh.TestEntity, string] { return subject },
			MakeEnt: psh.MakeTestEntity,
			MakeCtx: psh.MakeCtx,
		},
		comprotocontracts.OnePhaseCommitProtocol[psh.TestEntity, string]{
			Subject: func(tb testing.TB) comprotocontracts.OnePhaseCommitProtocolSubject[psh.TestEntity, string] {
				return comprotocontracts.OnePhaseCommitProtocolSubject[psh.TestEntity, string]{
					Resource:      subject,
					CommitManager: cm,
				}
			},
			MakeEnt: psh.MakeTestEntity,
			MakeCtx: psh.MakeCtx,
		},
		pubsubcontracts.Publisher[psh.TestEntity, string]{Subject: func(tb testing.TB) pubsubcontracts.PublisherSubject[psh.TestEntity, string] { return subject },
			MakeEnt: psh.MakeTestEntity,
			MakeCtx: psh.MakeCtx,
		},
		frmetacontracts.MetaAccessor[psh.TestEntity, string, string]{
			Subject: func(tb testing.TB) frmetacontracts.MetaAccessorSubject[psh.TestEntity, string, string] {
				return frmetacontracts.MetaAccessorSubject[psh.TestEntity, string, string]{
					MetaAccessor: subject,
					Resource:     subject,
					Publisher:    subject,
				}
			},
			MakeEnt: psh.MakeTestEntity,
			MakeCtx: psh.MakeCtx,
			MakeV:   psh.MakeString,
		},
	)
}

func TestStorage_contracts(t *testing.T) {
	s := testcase.NewSpec(t)
	storage := NewTestEntityStorage(t)

	contracts2.Contract[psh.TestEntity, string, string]{
		Subject: func(tb testing.TB) contracts2.ContractSubject[psh.TestEntity, string] {
			return contracts2.ContractSubject[psh.TestEntity, string]{
				MetaAccessor:  storage,
				CommitManager: storage,
				Resource:      storage,
			}
		},
		MakeEnt: psh.MakeTestEntity,
		MakeCtx: psh.MakeCtx,
		MakeV:   psh.MakeString,
	}.Spec(s)
}

func TestStorage_mappingHasSchemaInTableName(t *testing.T) {
	cm := postgresql.NewConnectionManager(psh.DatabaseURL(t))
	psh.MigrateTestEntity(t, cm)

	mapper := psh.TestEntityMapping()
	mapper.Table = `public.` + mapper.Table

	subject := NewTestEntityStorage(t)

	testcase.RunSuite(t, crudcontracts.Creator[psh.TestEntity, string]{
		Subject: func(tb testing.TB) crudcontracts.CreatorSubject[psh.TestEntity, string] { return subject },
		MakeCtx: psh.MakeCtx,
		MakeEnt: psh.MakeTestEntity,
	})
}
