package migration_test

import (
	"context"
	"errors"
	"testing"

	"go.llib.dev/frameless/adapter/memory"
	"go.llib.dev/frameless/port/migration"
)

func TestMigrator_Migrate(t *testing.T) {
	t.Run("successful migration", func(t *testing.T) {
		stateRepo := NewStateRepo()
		conn := NewConnection()
		migrator := &migration.Migrator[Connection]{
			Namespace:       "test-namespace",
			Connection:      conn,
			StateRepository: stateRepo,
			Steps: map[migration.Version]migration.Step[Connection]{
				"1": &stepMock{},
				"2": &stepMock{},
			},
		}

		err := migrator.Migrate(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("error handling", func(t *testing.T) {
		stateRepo := NewStateRepo()
		conn := NewConnection()
		migrator := &migration.Migrator[Connection]{
			Namespace:       "test-namespace",
			Connection:      conn,
			StateRepository: stateRepo,
			Steps: map[migration.Version]migration.Step[Connection]{
				"1": &stepMock{},
				"2": &errorStep{}, // simulate an error
			},
		}

		err := migrator.Migrate(context.Background())
		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("missing namespace", func(t *testing.T) {
		stateRepo := NewStateRepo()
		conn := NewConnection()
		migrator := &migration.Migrator[Connection]{
			Namespace:       "",
			Connection:      conn,
			StateRepository: stateRepo,
			Steps: map[migration.Version]migration.Step[Connection]{
				"1": &stepMock{},
			},
		}

		err := migrator.Migrate(context.Background())
		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("state repository error", func(t *testing.T) {
		stateRepo := &errorStateRepo{}
		conn := NewConnection()
		migrator := &migration.Migrator[Connection]{
			Namespace:       "test-namespace",
			Connection:      conn,
			StateRepository: stateRepo,
			Steps: map[migration.Version]migration.Step[Connection]{
				"1": &stepMock{},
			},
		}

		err := migrator.Migrate(context.Background())
		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("ensure state repository", func(t *testing.T) {
		stateRepo := NewStateRepo()
		conn := NewConnection()
		migrator := &migration.Migrator[Connection]{
			Namespace:       "test-namespace",
			Connection:      conn,
			StateRepository: stateRepo,
			EnsureStateRepository: func(ctx context.Context) error {
				return errors.New("simulated error")
			},
			Steps: map[migration.Version]migration.Step[Connection]{
				"1": &stepMock{},
			},
		}

		err := migrator.Migrate(context.Background())
		if err == nil {
			t.Errorf("expected error, got none")
		}
	})
}

type stepMock struct{}

func (s *stepMock) MigrateUp(Connection, context.Context) error   { return nil }
func (s *stepMock) MigrateDown(Connection, context.Context) error { return nil }

type errorStep struct{}

func (e *errorStep) MigrateUp(Connection, context.Context) error {
	return errors.New("simulated error")
}
func (e *errorStep) MigrateDown(Connection, context.Context) error { return nil }

type errorStateRepo struct{}

func (s *errorStateRepo) Create(context.Context, *migration.State) error {
	return errors.New("simulated error")
}
func (s *errorStateRepo) FindByID(context.Context, migration.StateID) (migration.State, bool, error) {
	return migration.State{}, false, errors.New("simulated error")
}
func (s *errorStateRepo) DeleteByID(context.Context, migration.StateID) error {
	return errors.New("simulated error")
}
func (s *errorStateRepo) BeginTx(context.Context) (context.Context, error) {
	return nil, errors.New("simulated error")
}
func (s *errorStateRepo) CommitTx(context.Context) error {
	return errors.New("simulated error")
}
func (s *errorStateRepo) RollbackTx(context.Context) error {
	return errors.New("simulated error")
}

func NewStateRepo() *memory.Repository[migration.State, migration.StateID] {
	return memory.NewRepository[migration.State, migration.StateID](memory.NewMemory())
}

type Connection struct {
	*memory.Memory
}

func NewConnection() Connection {
	return Connection{Memory: memory.NewMemory()}
}
