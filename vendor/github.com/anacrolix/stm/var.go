package stm

import (
	"sync"

	"github.com/alecthomas/atomic"
)

// Holds an STM variable.
type Var[T any] struct {
	value    atomic.Value[VarValue]
	watchers sync.Map
	mu       sync.Mutex
}

func (v *Var[T]) getValue() *atomic.Value[VarValue] {
	return &v.value
}

func (v *Var[T]) getWatchers() *sync.Map {
	return &v.watchers
}

func (v *Var[T]) getLock() *sync.Mutex {
	return &v.mu
}

func (v *Var[T]) changeValue(new any) {
	old := v.value.Load()
	newVarValue := old.Set(new)
	v.value.Store(newVarValue)
	if old.Changed(newVarValue) {
		go v.wakeWatchers(newVarValue)
	}
}

func (v *Var[T]) wakeWatchers(new VarValue) {
	v.watchers.Range(func(k, _ any) bool {
		tx := k.(*Tx)
		// We have to lock here to ensure that the Tx is waiting before we signal it. Otherwise we
		// could signal it before it goes to sleep and it will miss the notification.
		tx.mu.Lock()
		if read := tx.reads[v]; read != nil && read.Changed(new) {
			tx.cond.Broadcast()
			for !tx.waiting && !tx.completed {
				tx.cond.Wait()
			}
		}
		tx.mu.Unlock()
		return !v.value.Load().Changed(new)
	})
}

// Returns a new STM variable.
func NewVar[T any](val T) *Var[T] {
	v := &Var[T]{}
	v.value.Store(versionedValue[T]{
		value: val,
	})
	return v
}

func NewCustomVar[T any](val T, changed func(T, T) bool) *Var[T] {
	v := &Var[T]{}
	v.value.Store(customVarValue[T]{
		value:   val,
		changed: changed,
	})
	return v
}

func NewBuiltinEqVar[T comparable](val T) *Var[T] {
	return NewCustomVar(val, func(a, b T) bool {
		return a != b
	})
}
