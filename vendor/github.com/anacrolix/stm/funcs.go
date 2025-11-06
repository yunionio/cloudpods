package stm

import (
	"math/rand"
	"runtime/pprof"
	"sync"
	"time"
)

var (
	txPool = sync.Pool{New: func() any {
		expvars.Add("new txs", 1)
		tx := &Tx{
			reads:    make(map[txVar]VarValue),
			writes:   make(map[txVar]any),
			watching: make(map[txVar]struct{}),
		}
		tx.cond.L = &tx.mu
		return tx
	}}
	failedCommitsProfile *pprof.Profile
)

const (
	profileFailedCommits = false
	sleepBetweenRetries  = false
)

func init() {
	if profileFailedCommits {
		failedCommitsProfile = pprof.NewProfile("stmFailedCommits")
	}
}

func newTx() *Tx {
	tx := txPool.Get().(*Tx)
	tx.tries = 0
	tx.completed = false
	return tx
}

func WouldBlock[R any](fn Operation[R]) (block bool) {
	tx := newTx()
	tx.reset()
	_, block = catchRetry(fn, tx)
	if len(tx.watching) != 0 {
		panic("shouldn't have installed any watchers")
	}
	tx.recycle()
	return
}

// Atomically executes the atomic function fn.
func Atomically[R any](op Operation[R]) R {
	expvars.Add("atomically", 1)
	// run the transaction
	tx := newTx()
retry:
	tx.tries++
	tx.reset()
	if sleepBetweenRetries {
		shift := int64(tx.tries - 1)
		const maxShift = 30
		if shift > maxShift {
			shift = maxShift
		}
		ns := int64(1) << shift
		d := time.Duration(rand.Int63n(ns))
		if d > 100*time.Microsecond {
			tx.updateWatchers()
			time.Sleep(time.Duration(ns))
		}
	}
	tx.mu.Lock()
	ret, retry := catchRetry(op, tx)
	tx.mu.Unlock()
	if retry {
		expvars.Add("retries", 1)
		// wait for one of the variables we read to change before retrying
		tx.wait()
		goto retry
	}
	// verify the read log
	tx.lockAllVars()
	if tx.inputsChanged() {
		tx.unlock()
		expvars.Add("failed commits", 1)
		if profileFailedCommits {
			failedCommitsProfile.Add(new(int), 0)
		}
		goto retry
	}
	// commit the write log and broadcast that variables have changed
	tx.commit()
	tx.mu.Lock()
	tx.completed = true
	tx.cond.Broadcast()
	tx.mu.Unlock()
	tx.unlock()
	expvars.Add("commits", 1)
	tx.recycle()
	return ret
}

// AtomicGet is a helper function that atomically reads a value.
func AtomicGet[T any](v *Var[T]) T {
	return v.value.Load().Get().(T)
}

// AtomicSet is a helper function that atomically writes a value.
func AtomicSet[T any](v *Var[T], val T) {
	v.mu.Lock()
	v.changeValue(val)
	v.mu.Unlock()
}

// Compose is a helper function that composes multiple transactions into a
// single transaction.
func Compose[R any](fns ...Operation[R]) Operation[struct{}] {
	return VoidOperation(func(tx *Tx) {
		for _, f := range fns {
			f(tx)
		}
	})
}

// Select runs the supplied functions in order. Execution stops when a
// function succeeds without calling Retry. If no functions succeed, the
// entire selection will be retried.
func Select[R any](fns ...Operation[R]) Operation[R] {
	return func(tx *Tx) R {
		switch len(fns) {
		case 0:
			// empty Select blocks forever
			tx.Retry()
			panic("unreachable")
		case 1:
			return fns[0](tx)
		default:
			oldWrites := tx.writes
			tx.writes = make(map[txVar]any, len(oldWrites))
			for k, v := range oldWrites {
				tx.writes[k] = v
			}
			ret, retry := catchRetry(fns[0], tx)
			if retry {
				tx.writes = oldWrites
				return Select(fns[1:]...)(tx)
			} else {
				return ret
			}
		}
	}
}

type Operation[R any] func(*Tx) R

func VoidOperation(f func(*Tx)) Operation[struct{}] {
	return func(tx *Tx) struct{} {
		f(tx)
		return struct{}{}
	}
}

func AtomicModify[T any](v *Var[T], f func(T) T) {
	Atomically(VoidOperation(func(tx *Tx) {
		v.Set(tx, f(v.Get(tx)))
	}))
}
