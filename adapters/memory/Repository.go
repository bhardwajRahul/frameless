package memory

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/adamluzsi/frameless/pkg/reflects"
	"github.com/adamluzsi/frameless/ports/crud/extid"
	"github.com/adamluzsi/frameless/ports/iterators"
)

func NewRepository[Entity, ID any](m *Memory) *Repository[Entity, ID] {
	return &Repository[Entity, ID]{Memory: m}
}

func NewRepositoryWithNamespace[Entity, ID any](m *Memory, ns string) *Repository[Entity, ID] {
	return &Repository[Entity, ID]{Memory: m, Namespace: ns}
}

type Repository[Entity, ID any] struct {
	Memory        *Memory
	NewID         func(context.Context) (ID, error)
	Namespace     string
	initNamespace sync.Once
}

func (s *Repository[Entity, ID]) Create(ctx context.Context, ptr *Entity) error {
	if _, ok := extid.Lookup[ID](ptr); !ok {
		newID, err := s.MakeID(ctx)
		if err != nil {
			return err
		}

		if err := extid.Set(ptr, newID); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	id, _ := extid.Lookup[ID](ptr)
	if _, found, err := s.FindByID(ctx, id); err != nil {
		return err
	} else if found {
		return fmt.Errorf(`%T already exists with id: %v`, *new(Entity), id)
	}

	s.Memory.Set(ctx, s.GetNamespace(), s.IDToMemoryKey(id), base(ptr))

	return nil
}

func (s *Repository[Entity, ID]) FindByID(ctx context.Context, id ID) (_ent Entity, _found bool, _err error) {
	if err := ctx.Err(); err != nil {
		return _ent, false, err
	}
	if err := s.isDoneTx(ctx); err != nil {
		return _ent, false, err
	}

	ent, ok := s.Memory.Get(ctx, s.GetNamespace(), s.IDToMemoryKey(id))
	if !ok {
		return _ent, false, nil
	}
	return ent.(Entity), true, nil
}

func (s *Repository[Entity, ID]) FindAll(ctx context.Context) iterators.Iterator[Entity] {
	if err := ctx.Err(); err != nil {
		return iterators.Error[Entity](err)
	}
	if err := s.isDoneTx(ctx); err != nil {
		return iterators.Error[Entity](err)
	}
	return memoryAll[Entity](s.Memory, ctx, s.GetNamespace())
}

func (s *Repository[Entity, ID]) DeleteByID(ctx context.Context, id ID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.isDoneTx(ctx); err != nil {
		return err
	}
	if s.Memory.Del(ctx, s.GetNamespace(), s.IDToMemoryKey(id)) {
		return nil
	}
	return errNotFound(*new(Entity), id)
}

func (s *Repository[Entity, ID]) DeleteAll(ctx context.Context) error {
	iter := s.FindAll(ctx)
	defer iter.Close()
	for iter.Next() {
		id, _ := extid.Lookup[ID](iter.Value())
		_ = s.Memory.Del(ctx, s.GetNamespace(), s.IDToMemoryKey(id))
	}
	return iter.Err()
}

func (s *Repository[Entity, ID]) Update(ctx context.Context, ptr *Entity) error {
	id, ok := extid.Lookup[ID](ptr)
	if !ok {
		return fmt.Errorf(`entity doesn't have id field`)
	}

	_, found, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return errNotFound(*new(Entity), id)
	}

	s.Memory.Set(ctx, s.GetNamespace(), s.IDToMemoryKey(id), base(ptr))
	return nil
}

func (s *Repository[Entity, ID]) FindByIDs(ctx context.Context, ids ...ID) iterators.Iterator[Entity] {
	var m memoryActions = s.Memory
	if tx, ok := s.Memory.LookupTx(ctx); ok {
		m = tx
	}
	all := m.all(s.GetNamespace())
	var vs = make(map[string]Entity, len(ids))
	for _, id := range ids {
		key := s.IDToMemoryKey(id)
		v, ok := all[key]
		if !ok {
			return iterators.Error[Entity](errNotFound(*new(Entity), id))
		}
		vs[key] = v.(Entity)
	}
	return iterators.Slice[Entity](toSlice[Entity, string](vs))
}

func (s *Repository[Entity, ID]) Upsert(ctx context.Context, ptrs ...*Entity) error {
	var m memoryActions = s.Memory
	if tx, ok := s.Memory.LookupTx(ctx); ok {
		m = tx
	}
	for _, ptr := range ptrs {
		id, ok := extid.Lookup[ID](ptr)
		if !ok {
			nid, err := s.MakeID(ctx)
			if err != nil {
				return err
			}
			id = nid
			if err := extid.Set(ptr, nid); err != nil {
				return err
			}
		}
		key := s.IDToMemoryKey(id)
		m.set(s.GetNamespace(), key, base(ptr))
	}
	return nil
}

func (s *Repository[Entity, ID]) MakeID(ctx context.Context) (ID, error) {
	if s.NewID != nil {
		return s.NewID(ctx)
	}
	iid, err := newDummyID(*new(ID))
	if err != nil {
		return *new(ID), err
	}
	id, ok := iid.(ID)
	if !ok {
		return *new(ID), fmt.Errorf("newID failed")
	}
	return id, nil
}

func (s *Repository[Entity, ID]) IDToMemoryKey(id any) string {
	return fmt.Sprintf(`%#v`, id)
}

func (s *Repository[Entity, ID]) GetNamespace() string {
	s.initNamespace.Do(func() {
		if 0 < len(s.Namespace) {
			return
		}
		s.Namespace = reflects.FullyQualifiedName(*new(Entity))
	})
	return s.Namespace
}

func (s *Repository[Entity, ID]) getV(ptr interface{}) interface{} {
	return reflects.BaseValueOf(ptr).Interface()
}

func (s *Repository[Entity, ID]) isDoneTx(ctx context.Context) error {
	tx, ok := s.Memory.LookupTx(ctx)
	if !ok {
		return nil
	}
	if tx.done {
		return errTxDone
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func NewMemory() *Memory {
	return &Memory{}
}

type Memory struct {
	m      sync.Mutex
	tables map[string]MemoryNamespace

	ns struct {
		init  sync.Once
		value string
	}
}

type (
	ctxKeyMemoryMeta   struct{ NS string }
	ctxValueMemoryMeta map[string]interface{}
)

func (m *Memory) ctxKeyMeta() ctxKeyMemoryMeta {
	return ctxKeyMemoryMeta{NS: m.getNS()}
}

func (m *Memory) lookupMetaMap(ctx context.Context) (ctxValueMemoryMeta, bool) {
	if ctx == nil {
		return nil, false
	}
	mm, ok := ctx.Value(m.ctxKeyMeta()).(ctxValueMemoryMeta)
	return mm, ok
}

func (m *Memory) SetMeta(ctx context.Context, key string, value interface{}) (context.Context, error) {
	if ctx == nil {
		return ctx, fmt.Errorf(`input context.Context was nil`)
	}
	mm, ok := m.lookupMetaMap(ctx)
	if !ok {
		mm = make(ctxValueMemoryMeta)
		ctx = context.WithValue(ctx, m.ctxKeyMeta(), m)
	}
	mm[key] = base(value)
	return ctx, nil
}

func (m *Memory) LookupMeta(ctx context.Context, key string, ptr interface{}) (_found bool, _err error) {
	if ctx == nil {
		return false, nil
	}
	mm, ok := m.lookupMetaMap(ctx)
	if !ok {
		return false, nil
	}
	v, ok := mm[key]
	if !ok {
		return false, nil
	}
	return true, reflects.Link(v, ptr)
}

type memoryActions interface {
	all(namespace string) map[string]interface{}
	get(namespace string, key string) (interface{}, bool)
	set(namespace, key string, value interface{})
	del(namespace string, key string) bool
}

func base(ent any) interface{} {
	return reflects.BaseValueOf(ent).Interface()
}

func (m *Memory) Get(ctx context.Context, namespace string, key string) (interface{}, bool) {
	if tx, ok := m.LookupTx(ctx); ok && !tx.done {
		return tx.get(namespace, key)
	}
	return m.get(namespace, key)
}

func memoryAll[Entity any](m *Memory, ctx context.Context, namespace string) iterators.Iterator[Entity] {
	var T Entity
	return iterators.Slice[Entity](m.All(T, ctx, namespace).([]Entity))
}

func (m *Memory) All(T any, ctx context.Context, namespace string) (sliceOfT interface{}) {
	if tx, ok := m.LookupTx(ctx); ok && !tx.done {
		return m.toTSlice(T, tx.all(namespace))
	}
	return m.toTSlice(T, m.all(namespace))
}

func (m *Memory) toTSlice(T any, vs map[string]interface{}) interface{} {
	rslice := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(T)), 0, len(vs))
	for _, v := range vs {
		rslice = reflect.Append(rslice, reflect.ValueOf(v))
	}
	return rslice.Interface()
}

func (m *Memory) all(namespace string) map[string]interface{} {
	m.m.Lock()
	defer m.m.Unlock()
	var vs = make(map[string]interface{})
	for k, v := range m.namespace(namespace) {
		vs[k] = v
	}
	return vs
}

func (m *Memory) get(namespace string, key string) (interface{}, bool) {
	m.m.Lock()
	defer m.m.Unlock()
	ns := m.namespace(namespace)
	v, ok := ns[key]
	return v, ok
}

func (m *Memory) Set(ctx context.Context, namespace, key string, value interface{}) {
	if tx, ok := m.LookupTx(ctx); ok {
		tx.set(namespace, key, value)
		return
	}

	m.set(namespace, key, value)
}

func (m *Memory) set(namespace, key string, value interface{}) {
	m.m.Lock()
	defer m.m.Unlock()
	tbl := m.namespace(namespace)
	tbl[key] = base(value)
	return
}

func (m *Memory) Del(ctx context.Context, namespace string, key string) bool {
	if tx, ok := m.LookupTx(ctx); ok {
		return tx.del(namespace, key)
	}
	return m.del(namespace, key)
}

func (m *Memory) del(namespace string, key string) bool {
	m.m.Lock()
	defer m.m.Unlock()
	ns := m.namespace(namespace)
	if _, ok := ns[key]; !ok {
		return false
	}
	delete(ns, key)
	return true
}

type MemoryNamespace map[string]interface{}

func (m *Memory) namespace(name string) MemoryNamespace {
	if m.tables == nil {
		m.tables = make(map[string]MemoryNamespace)
	}
	if _, ok := m.tables[name]; !ok {
		m.tables[name] = make(MemoryNamespace)
	}
	return m.tables[name]
}

type MemoryTx struct {
	m       sync.Mutex
	done    bool
	super   memoryActions
	changes map[string]memoryTxChanges
}

type memoryTxChanges struct {
	Values  MemoryNamespace
	Deleted map[string]struct{}
}

func (tx *MemoryTx) all(namespace string) map[string]interface{} {
	tx.m.Lock()
	defer tx.m.Unlock()
	svs := tx.super.all(namespace)
	cvs := tx.getChanges(namespace)
	avs := make(map[string]interface{})
	for k, v := range svs {
		avs[k] = v
	}
	for k, _ := range cvs.Deleted {
		delete(avs, k)
	}
	for k, v := range cvs.Values {
		avs[k] = v
	}
	return avs
}

func (tx *MemoryTx) get(namespace string, key string) (interface{}, bool) {
	tx.m.Lock()
	defer tx.m.Unlock()
	changes := tx.getChanges(namespace)
	v, ok := changes.Values[key]
	if ok {
		return v, ok
	}
	if _, isDeleted := changes.Deleted[key]; isDeleted {
		return nil, false
	}
	return tx.super.get(namespace, key)
}

func (tx *MemoryTx) set(namespace, key string, value interface{}) {
	tx.m.Lock()
	defer tx.m.Unlock()
	tx.getChanges(namespace).Values[key] = value
}

func (tx *MemoryTx) del(namespace, key string) bool {
	if _, ok := tx.get(namespace, key); !ok {
		return false
	}
	tx.m.Lock()
	defer tx.m.Unlock()
	changes := tx.getChanges(namespace)
	delete(changes.Values, key)
	changes.Deleted[key] = struct{}{}
	return true
}

func (tx *MemoryTx) commit() error {
	if tx.done {
		return errTxDone
	}
	tx.m.Lock()
	defer tx.m.Unlock()
	tx.done = true
	for namespace, values := range tx.changes {
		for key, _ := range values.Deleted {
			tx.super.del(namespace, key)
		}
		for key, value := range values.Values {
			tx.super.set(namespace, key, value)
		}
	}
	return nil
}

func (tx *MemoryTx) rollback() error {
	if tx.done {
		return errTxDone
	}
	tx.done = true
	super, ok := tx.super.(*MemoryTx)
	if !ok {
		return nil
	}
	// We rollback recursively because most resource don't support multi level transactions
	// and I would like to avoid supporting it here until I have proper use-case with it.
	// Then adding a flag to turn off this behavior should be easy-peasy.
	return super.rollback()
}

func (tx *MemoryTx) getChanges(name string) memoryTxChanges {
	if tx.changes == nil {
		tx.changes = make(map[string]memoryTxChanges)
	}
	if _, ok := tx.changes[name]; !ok {
		tx.changes[name] = memoryTxChanges{
			Values:  make(MemoryNamespace),
			Deleted: make(map[string]struct{}),
		}
	}
	return tx.changes[name]
}

func (m *Memory) getNS() string {
	m.ns.init.Do(func() { m.ns.value = genStringUID() })
	return m.ns.value
}

type ctxKeyMemoryTx struct{ NS string }

func (m *Memory) ctxKeyMemoryTx() ctxKeyMemoryTx {
	return ctxKeyMemoryTx{NS: m.getNS()}
}

func (m *Memory) BeginTx(ctx context.Context) (context.Context, error) {
	var super memoryActions = m
	if tx, ok := m.LookupTx(ctx); ok {
		super = tx
	}
	return context.WithValue(ctx, m.ctxKeyMemoryTx(), &MemoryTx{super: super}), nil
}

func (m *Memory) CommitTx(ctx context.Context) error {
	if tx, ok := m.LookupTx(ctx); ok {
		return tx.commit()
	}
	return errNoTx
}

func (m *Memory) RollbackTx(ctx context.Context) error {
	if tx, ok := m.LookupTx(ctx); ok {
		return tx.rollback()
	}
	return errNoTx
}

func (m *Memory) LookupTx(ctx context.Context) (*MemoryTx, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(m.ctxKeyMemoryTx()).(*MemoryTx)
	return tx, ok
}