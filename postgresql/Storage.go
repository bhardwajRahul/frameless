package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/adamluzsi/frameless/extid"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/adamluzsi/frameless"
	"github.com/adamluzsi/frameless/iterators"

	"github.com/lib/pq"
)

func NewStorage(T interface{}, cm *ConnectionManager, m Mapping) *Storage {
	return &Storage{
		T:                 T,
		ConnectionManager: cm,
		Mapping:           m,
	}
}

// Storage is a frameless external resource supplier to store a certain entity type.
//
// SRP: DBA
type Storage struct {
	T                 interface{}
	ConnectionManager *ConnectionManager
	Mapping           Mapping
}

func (pg *Storage) Create(ctx context.Context, ptr interface{}) (rErr error) {
	query := fmt.Sprintf("INSERT INTO %s (%s)\n", pg.Mapping.TableName(), pg.queryColumnList())
	query += fmt.Sprintf("VALUES (%s)\n", pg.queryColumnPlaceHolders(pg.newPrepareStatementPlaceholderGenerator()))

	ctx, td, err := pg.withTx(ctx)
	if err != nil {
		return err
	}
	defer func() { rErr = td(rErr) }()

	c, err := pg.ConnectionManager.GetConnection(ctx)
	if err != nil {
		return err
	}

	if _, ok := extid.Lookup(ptr); !ok {
		// TODO: add serialize TX level here

		id, err := pg.Mapping.NewID(ctx)
		if err != nil {
			return err
		}

		if err := extid.Set(ptr, id); err != nil {
			return err
		}
	}

	args, err := pg.Mapping.ToArgs(ptr)
	if err != nil {
		return err
	}
	if _, err := c.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return pg.notify(ctx, c, pg.getCreateSubscriptionName(), ptr)
}

func (pg *Storage) FindByID(ctx context.Context, ptr, id interface{}) (bool, error) {
	c, err := pg.ConnectionManager.GetConnection(ctx)
	if err != nil {
		return false, err
	}

	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %q = $1`, pg.queryColumnList(), pg.Mapping.TableName(), pg.Mapping.IDName())

	err = pg.Mapping.Map(c.QueryRowContext(ctx, query, id), ptr)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (pg *Storage) DeleteAll(ctx context.Context) (rErr error) {
	ctx, td, err := pg.withTx(ctx)
	if err != nil {
		return err
	}
	defer func() { rErr = td(rErr) }()

	c, err := pg.ConnectionManager.GetConnection(ctx)
	if err != nil {
		return err
	}

	var (
		tableName = pg.Mapping.TableName()
		topicName = pg.getDeleteAllSubscriptionName()
		message   = reflect.New(reflect.TypeOf(pg.T)).Elem().Interface()
		query     = fmt.Sprintf(`DELETE FROM %s`, tableName)
	)

	if _, err := c.ExecContext(ctx, query); err != nil {
		return err
	}

	if message != nil {
		if err := pg.notify(ctx, c, topicName, message); err != nil {
			return err
		}
	}

	return nil
}

func (pg *Storage) DeleteByID(ctx context.Context, id interface{}) (rErr error) {
	var (
		sid       = id.(string)
		query     = fmt.Sprintf(`DELETE FROM %s WHERE "id" = $1`, pg.Mapping.TableName())
		topicName = pg.getDeleteByIDSubscriptionName()
		message   = reflect.New(reflect.TypeOf(pg.T)).Interface()
	)

	if err := extid.Set(message, sid); err != nil {
		return err
	}

	ctx, td, err := pg.withTx(ctx)
	if err != nil {
		return err
	}
	defer func() { rErr = td(rErr) }()

	c, err := pg.ConnectionManager.GetConnection(ctx)
	if err != nil {
		return err
	}

	result, err := c.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf(`ErrNotFound`)
	}

	if message != nil {
		if err := pg.notify(ctx, c, topicName, message); err != nil {
			return err
		}
	}

	return nil
}

func (pg *Storage) newPrepareStatementPlaceholderGenerator() func() string {
	var index = 0
	return func() string {
		index++
		return fmt.Sprintf(`$%d`, index)
	}
}

func (pg *Storage) Update(ctx context.Context, ptr interface{}) (rErr error) {
	args, err := pg.Mapping.ToArgs(ptr)
	if err != nil {
		return err
	}

	var (
		query           = fmt.Sprintf("UPDATE %s", pg.Mapping.TableName())
		nextPlaceHolder = pg.newPrepareStatementPlaceholderGenerator()
		idPlaceHolder   = nextPlaceHolder()
		querySetParts   []string
	)
	for _, name := range pg.Mapping.ColumnNames() {
		querySetParts = append(querySetParts, fmt.Sprintf(`%q = %s`, name, nextPlaceHolder()))
	}
	if len(querySetParts) > 0 {
		query += fmt.Sprintf("\nSET %s", strings.Join(querySetParts, `, `))
	}
	query += fmt.Sprintf("\nWHERE %q = %s", pg.Mapping.IDName(), idPlaceHolder)

	id, ok := extid.Lookup(ptr)
	if !ok {
		return fmt.Errorf(`missing entity id`)
	}

	args = append([]interface{}{id}, args...)

	ctx, td, err := pg.withTx(ctx)
	if err != nil {
		return err
	}
	defer func() { rErr = td(rErr) }()

	c, err := pg.ConnectionManager.GetConnection(ctx)
	if err != nil {
		return err
	}

	if res, err := c.ExecContext(ctx, query, args...); err != nil {
		return err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf(`deployment environment not found`)
		}
	}

	return pg.notify(ctx, c, pg.getUpdateSubscriptionName(), ptr)
}

func (pg *Storage) FindAll(ctx context.Context) iterators.Interface {
	query := fmt.Sprintf(`SELECT %s FROM %s`, pg.queryColumnList(), pg.Mapping.TableName())

	c, err := pg.ConnectionManager.GetConnection(ctx)
	if err != nil {
		return iterators.NewError(err)
	}

	rows, err := c.QueryContext(ctx, query)
	if err != nil {
		return iterators.NewError(err)
	}

	return iterators.NewSQLRows(rows, pg.Mapping)
}

func (pg *Storage) BeginTx(ctx context.Context) (context.Context, error) {
	return pg.ConnectionManager.BeginTx(ctx)
}

func (pg *Storage) CommitTx(ctx context.Context) error {
	return pg.ConnectionManager.CommitTx(ctx)
}

func (pg *Storage) RollbackTx(ctx context.Context) error {
	return pg.ConnectionManager.RollbackTx(ctx)
}

func (pg *Storage) withTx(ctx context.Context) (context.Context, func(error) error, error) {
	tx, err := pg.BeginTx(ctx)
	if err != nil {
		return nil, nil, err
	}

	return tx, func(rErr error) error {
		if rErr != nil {
			_ = pg.RollbackTx(tx)
			return rErr
		}

		return pg.CommitTx(tx)
	}, nil
}

func (pg *Storage) queryColumnPlaceHolders(nextPlaceholder func() string) string {
	var phs []string
	for range pg.Mapping.ColumnNames() {
		phs = append(phs, nextPlaceholder())
	}
	return strings.Join(phs, `, `)
}

func (pg *Storage) queryColumnList() string {
	var (
		src = pg.Mapping.ColumnNames()
		dst = make([]string, 0, len(src))
	)
	for _, name := range src {
		dst = append(dst, fmt.Sprintf(`%s`, name))
	}
	return strings.Join(dst, `, `)
}

//--------------------------------------------------------------------------------------------------------------------//

func (pg *Storage) notify(ctx context.Context, tx Connection, name string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `SELECT pg_notify($1, $2)`, name, string(data))
	return err
}

func (pg *Storage) newPostgresSubscription(connstr string, name string, subscriber frameless.Subscriber) (*postgresCommonSubscription, error) {
	const (
		minReconnectInterval = 10 * time.Second
		maxReconnectInterval = time.Minute
	)
	var sub postgresCommonSubscription
	sub.T = pg.T
	sub.rType = reflect.TypeOf(pg.T)
	sub.subscriber = subscriber
	sub.listener = pq.NewListener(connstr, minReconnectInterval, maxReconnectInterval, sub.reportProblemToSubscriber)
	sub.exit.context, sub.exit.signaler = context.WithCancel(context.Background())
	return &sub, sub.start(name)
}

type postgresCommonSubscription struct {
	T          interface{}
	rType      reflect.Type
	subscriber frameless.Subscriber
	listener   *pq.Listener
	exit       struct {
		wg       sync.WaitGroup
		context  context.Context
		signaler func()
	}
}

func (sub *postgresCommonSubscription) start(name string) error {
	if err := sub.listener.Listen(name); err != nil {
		return err
	}

	sub.exit.wg.Add(1)
	go sub.handler()
	return nil
}

func (sub *postgresCommonSubscription) handler() {
	defer sub.exit.wg.Done()

wrk:
	for {
		ctx := context.Background()

		select {
		case <-sub.exit.context.Done():
			break wrk

		case n := <-sub.listener.Notify:
			ptr := reflect.New(sub.rType)

			if sub.handleError(ctx, json.Unmarshal([]byte(n.Extra), ptr.Interface())) {
				continue wrk
			}

			sub.handleError(ctx, sub.subscriber.Handle(ctx, ptr.Elem().Interface()))

			continue wrk
		case <-time.After(time.Minute):
			sub.handleError(ctx, sub.listener.Ping())
			continue wrk
		}
	}
}

func (sub *postgresCommonSubscription) handleError(ctx context.Context, err error) (isErrorHandled bool) {
	if err == nil {
		return false
	}

	if sErr := sub.subscriber.Error(ctx, err); sErr != nil {
		log.Println(`ERROR`, sErr.Error())
	}

	return true
}

func (sub *postgresCommonSubscription) Close() error {
	if sub.exit.signaler == nil || sub.listener == nil {
		return nil
	}

	sub.exit.signaler()
	sub.exit.wg.Wait()
	return sub.listener.Close()
}

func (sub *postgresCommonSubscription) reportProblemToSubscriber(_ pq.ListenerEventType, err error) {
	if err != nil {
		_ = sub.subscriber.Error(context.Background(), err)
	}
}

func (pg *Storage) SubscribeToCreate(_ context.Context, subscriber frameless.Subscriber) (frameless.Subscription, error) {
	return pg.newPostgresSubscription(pg.ConnectionManager.DSN, pg.getCreateSubscriptionName(), subscriber)
}

func (pg *Storage) getCreateSubscriptionName() string {
	return `create_` + pg.Mapping.TableName()
}

func (pg *Storage) SubscribeToUpdate(_ context.Context, subscriber frameless.Subscriber) (frameless.Subscription, error) {
	return pg.newPostgresSubscription(pg.ConnectionManager.DSN, pg.getUpdateSubscriptionName(), subscriber)
}

func (pg *Storage) getUpdateSubscriptionName() string {
	return `update_` + pg.Mapping.TableName()
}

func (pg *Storage) SubscribeToDeleteByID(_ context.Context, subscriber frameless.Subscriber) (frameless.Subscription, error) {
	return pg.newPostgresSubscription(pg.ConnectionManager.DSN, pg.getDeleteByIDSubscriptionName(), subscriber)
}

func (pg *Storage) getDeleteByIDSubscriptionName() string {
	return `delete_by_id_` + pg.Mapping.TableName()
}

func (pg *Storage) SubscribeToDeleteAll(_ context.Context, subscriber frameless.Subscriber) (frameless.Subscription, error) {
	return pg.newPostgresSubscription(pg.ConnectionManager.DSN, pg.getDeleteAllSubscriptionName(), subscriber)
}

func (pg *Storage) getDeleteAllSubscriptionName() string {
	return `delete_all_` + pg.Mapping.TableName()
}
