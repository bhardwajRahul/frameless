package frameless

import (
	"context"
	"fmt"
)

func Recover(errp *error) {
	r := recover()
	if r != nil {
		switch r := r.(type) {
		case error:
			*errp = r
		default:
			*errp = fmt.Errorf("%v", r)
		}
	}
}

func FinishTx(errp *error, commit, rollback func() error) {
	if errp == nil {
		panic(fmt.Errorf(`error pointer cannot be nil for Finish Tx methods`))
	}
	if *errp != nil {
		_ = rollback()
		return
	}
	*errp = commit()
}

func FinishOnePhaseCommit(errp *error, cm OnePhaseCommitProtocol, tx context.Context) {
	FinishTx(errp, func() error {
		return cm.CommitTx(tx)
	}, func() error {
		return cm.RollbackTx(tx)
	})
}