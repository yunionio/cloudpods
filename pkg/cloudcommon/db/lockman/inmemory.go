package lockman

import (
	"context"
	"sync"

	"yunion.io/x/log"
	"runtime/debug"
)

const (
	debug_log = false
)

/*type SInMemoryLockOwner struct {
	owner context.Context
}*/

type SInMemoryLockRecord struct {
	lock   *sync.Mutex
	cond   *sync.Cond
	holder context.Context
	depth  int
	waiter *FIFO
}

func newInMemoryLockRecord(ctx context.Context) *SInMemoryLockRecord {
	lock := &sync.Mutex{}
	cond := sync.NewCond(lock)
	rec := SInMemoryLockRecord{lock: lock, cond: cond, holder: ctx, depth: 0, waiter: NewFIFO()}
	return &rec
}

func (rec *SInMemoryLockRecord) lockContext(ctx context.Context) {
	rec.lock.Lock()
	defer rec.lock.Unlock()

	if rec.holder == nil {
		rec.holder = ctx
		rec.depth = 1
		return
	}

	if debug_log {
		log.Debugf("rec.hold=[%p] ctx=[%p] %v", rec.holder, ctx, rec.holder==ctx)
	}

	if rec.holder == ctx {
		rec.depth += 1
		if debug_log {
			log.Infof("lockContext: same ctx, depth: %d [%p]", rec.depth, rec.holder)
		}
		if rec.depth > 32 {
			// XXX MUST BE BUG ???
			debug.PrintStack()
			panic("Too many recursive locks!!!")
		}
		return
	}

	// check
	rec.waiter.Enum(func(ele interface{}) {
		electx := ele.(context.Context)
		if electx == ctx {
			log.Fatalf("try to lock from a waiter context????")
		}
	})

	rec.waiter.Push(ctx)

	if debug_log {
		log.Debugf("waiter size %d after push", rec.waiter.Len())
		log.Debugf("Start to wait ... [%p]", ctx)
	}

	for rec.holder != nil {
		rec.cond.Wait()
	}

	if debug_log {
		log.Debugf("End of wait ... [%p]", ctx)
	}

	rec.waiter.Pop(ctx)

	if debug_log {
		log.Debugf("waiter size %d after pop", rec.waiter.Len())
	}

	rec.holder = ctx
	rec.depth = 1
}

func (rec *SInMemoryLockRecord) unlockContext(ctx context.Context) (needClean bool) {
	rec.lock.Lock()
	defer rec.lock.Unlock()

	if rec.holder != ctx {
		log.Fatalf("try to unlock a wait context???")
	}

	if debug_log {
		log.Debugf("unlockContext depth %d [%p]", rec.depth, ctx)
	}

	rec.depth -= 1

	if rec.depth <= 0 {
		if debug_log {
			log.Debugf("depth 0, to release lock for context [%p]", ctx)
		}

		rec.holder = nil
		if rec.waiter.Len() == 0 {
			return true
		}
		rec.cond.Signal()
	}

	return false
}

type SInMemoryLockManager struct {
	tableLock *sync.Mutex
	lockTable map[string]*SInMemoryLockRecord
}

func NewInMemoryLockManager() ILockManager {
	lockMan := SInMemoryLockManager{tableLock: &sync.Mutex{}, lockTable: make(map[string]*SInMemoryLockRecord)}
	return &lockMan
}

func (lockman *SInMemoryLockManager) getRecordWithLock(ctx context.Context, key string) *SInMemoryLockRecord {
	lockman.tableLock.Lock()
	defer lockman.tableLock.Unlock()

	return lockman.getRecord(ctx, key, true)
}

func (lockman *SInMemoryLockManager) getRecord(ctx context.Context, key string, new bool) *SInMemoryLockRecord {
	_, ok := lockman.lockTable[key]
	if !ok {
		if !new {
			return nil
		}
		lockman.lockTable[key] = newInMemoryLockRecord(ctx)
	}
	return lockman.lockTable[key]
}

func (lockman *SInMemoryLockManager) LockKey(ctx context.Context, key string) {
	record := lockman.getRecordWithLock(ctx, key)

	record.lockContext(ctx)
}

func (lockman *SInMemoryLockManager) UnlockKey(ctx context.Context, key string) {
	lockman.tableLock.Lock()
	defer lockman.tableLock.Unlock()

	record := lockman.getRecord(ctx, key, false)
	if record == nil {
		log.Warningf("unlock an none exist lock????")
		return
	}

	needClean := record.unlockContext(ctx)
	if needClean {
		delete(lockman.lockTable, key)
	}
}