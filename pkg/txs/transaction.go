package txs

import (
	"context"
	"github.com/adamluzsi/frameless/pkg/errutils"
	"github.com/adamluzsi/frameless/pkg/teardown"
)

type transaction struct {
	parent    *transaction
	done      bool
	context   context.Context
	cancel    func()
	rollbacks teardown.Teardown
}

func (tx *transaction) OnRollback(fn func(context.Context) error) error {
	if tx.done {
		return ErrTxDone
	}
	tx.rollbacks.Defer(func() error {
		return fn(tx.context)
	})
	return nil
}

func (tx *transaction) Commit() error {
	if err := tx.finish(); err != nil {
		return err
	}
	if tx.parent != nil {
		return tx.parent.OnRollback(func(context.Context) error {
			return tx.rollbacks.Finish()
		})
	}
	return nil
}

func (tx *transaction) Rollback() (rErr error) {
	if tx.done {
		return ErrTxDone
	}
	defer func() { rErr = errutils.Merge(rErr, tx.finish()) }()
	defer func() {
		if tx.parent == nil {
			return
		}
		rErr = errutils.Merge(rErr, tx.parent.Rollback())
	}()
	return tx.rollbacks.Finish()
}

func (tx *transaction) isDone() bool {
	return tx.done
}

func (tx *transaction) finish() error {
	if tx.done {
		return ErrTxDone
	}
	tx.done = true
	tx.cancel()
	return nil
}
