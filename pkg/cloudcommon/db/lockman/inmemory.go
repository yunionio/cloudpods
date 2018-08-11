package lockman

import (
	"context"
	"runtime/debug"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/fifoutils"
)

type SInMemoryLockOwner struct {
	owner context.Context
	ready chan bool
}

type SInMemoryLockRecord struct {
	lock    *sync.Mutex
	holder  context.Context
	counter int
	queue   *fifoutils.FIFO
}

func newInMemoryLockRecord(ctx context.Context) *SInMemoryLockRecord {
	rec := SInMemoryLockRecord{lock: &sync.Mutex{}, queue: fifoutils.NewFIFO(), holder: ctx, counter: 0}
	return &rec
}

func (rec *SInMemoryLockRecord) lockContext(ctx context.Context) *SInMemoryLockOwner {
	rec.lock.Lock()
	defer rec.lock.Unlock()

	if rec.holder == ctx {
		rec.counter += 1
		log.Infof("lockContext: same ctx, counter: %d [%p]", rec.counter, rec.holder)
		if rec.counter > 32 {
			// MUST BE BUG
			debug.PrintStack()
			panic("Too many recursive locks!!!")
		}
		return nil
	}
	// check
	for i := 0; i < rec.queue.Len(); i += 1 {
		ele := rec.queue.ElementAt(i).(*SInMemoryLockOwner)
		if ele.owner == ctx {
			log.Fatalf("try to lock from a wait context????")
		}
	}
	owner := SInMemoryLockOwner{owner: ctx, ready: make(chan bool)}
	rec.queue.Push(&owner)

	return &owner
}

func (rec *SInMemoryLockRecord) unlockContext(ctx context.Context) (needClean bool) {
	rec.lock.Lock()
	defer rec.lock.Unlock()

	if rec.holder != ctx {
		log.Fatalf("try to unlock a wait context???")
	}

	rec.counter -= 1

	if rec.counter <= 0 {
		if rec.queue.Len() == 0 {
			return true
		}
		newHolder := rec.queue.Pop().(*SInMemoryLockOwner)
		rec.holder = newHolder.owner
		rec.counter = 1
		newHolder.notify()
	}

	return false
}

func (owner *SInMemoryLockOwner) wait() {
	// log.Infof("wait for notify %p", owner.owner)
	<-owner.ready
}

func (owner *SInMemoryLockOwner) notify() {
	// log.Infof("notify %p", owner.owner)
	owner.ready <- true
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

	owner := record.lockContext(ctx)
	if owner != nil {
		owner.wait()
	}
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
