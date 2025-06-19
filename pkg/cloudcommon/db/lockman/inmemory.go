// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lockman

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/petermattis/goid"

	"yunion.io/x/log"
)

var (
	debug_log = false
)

/*type SInMemoryLockOwner struct {
	owner context.Context
}*/

type SInMemoryLockRecord struct {
	key    string
	lock   *sync.Mutex
	cond   *sync.Cond
	holder int64
	depth  int
	waiter *FIFO
}

func newInMemoryLockRecord(ctxDummy context.Context) *SInMemoryLockRecord {
	lock := &sync.Mutex{}
	cond := sync.NewCond(lock)
	rec := SInMemoryLockRecord{lock: lock, cond: cond, holder: -1, depth: 0, waiter: NewFIFO()}
	return &rec
}

func (rec *SInMemoryLockRecord) fatalf(fmtStr string, args ...interface{}) {
	debug.PrintStack()
	log.Fatalf(fmtStr, args...)
}

func (rec *SInMemoryLockRecord) lockContext(ctxDummy context.Context) {
	rec.lock.Lock()
	defer rec.lock.Unlock()

	curGoid := goid.Get()

	if rec.holder < 0 {
		if debug_log {
			log.Debugf("lockContext: curGoid=[%d] key=[%s] create new record", curGoid, rec.key)
		}
		rec.holder = curGoid
		rec.depth = 1
		return
	}

	if debug_log {
		log.Debugf("rec.hold=[%d] ctx=[%d] %v key=[%s]", rec.holder, curGoid, rec.holder == curGoid, rec.key)
	}

	if rec.holder == curGoid {
		rec.depth += 1
		if debug_log {
			log.Infof("lockContext: same ctx, depth: %d holder=[%d] ctx=[%d] key=[%s]", rec.depth, rec.holder, curGoid, rec.key)
		}
		if rec.depth > 32 {
			// XXX MUST BE BUG ???
			rec.fatalf("Too many recursive locks!!! key=[%s]", rec.key)
		}
		return
	}

	// check
	rec.waiter.Enum(func(ele interface{}) {
		electx := ele.(int64)
		if electx == curGoid {
			rec.fatalf("try to lock from a waiter context???? curGoid=[%d] waiterGoid=[%d] key=[%s]", curGoid, electx, rec.key)
		}
	})

	rec.waiter.Push(curGoid)

	if debug_log {
		log.Debugf("waiter size %d after push curGoid=[%d]", rec.waiter.Len(), curGoid)
		log.Debugf("Start to wait ... holder=[%d] curGoid [%d] key=[%s]", rec.holder, curGoid, rec.key)
	}

	for rec.holder >= 0 {
		rec.cond.Wait()
	}

	if debug_log {
		log.Debugf("End of wait ... holder=[%d] curGoid [%d] key=[%s]", rec.holder, curGoid, rec.key)
	}

	rec.waiter.Pop(curGoid)

	if debug_log {
		log.Debugf("waiter size %d after pop curGoid=[%d] key=[%s]", rec.waiter.Len(), curGoid, rec.key)
	}

	rec.holder = curGoid
	rec.depth = 1
}

func (rec *SInMemoryLockRecord) unlockContext(ctxDummy context.Context) (needClean bool) {
	rec.lock.Lock()
	defer rec.lock.Unlock()

	curGoid := goid.Get()

	if rec.holder != curGoid {
		rec.fatalf("try to unlock a wait context??? key=[%s] holder=[%d] curGoid=[%d]", rec.key, rec.holder, curGoid)
	}

	if debug_log {
		log.Debugf("unlockContext depth %d curGoid=[%d] key=[%s]", rec.depth, curGoid, rec.key)
	}

	rec.depth -= 1

	if rec.depth <= 0 {
		if debug_log {
			log.Debugf("depth 0, to release lock for context curGoid=[%d] key=[%s]", curGoid, rec.key)
		}

		rec.holder = -1
		if rec.waiter.Len() == 0 {
			return true
		}
		rec.cond.Signal()
	}

	return false
}

type SInMemoryLockManager struct {
	*SBaseLockManager
	tableLock *sync.Mutex
	lockTable map[string]*SInMemoryLockRecord
}

func NewInMemoryLockManager() ILockManager {
	lockMan := SInMemoryLockManager{
		tableLock: &sync.Mutex{},
		lockTable: make(map[string]*SInMemoryLockRecord),
	}
	lockMan.SBaseLockManager = NewBaseLockManger(&lockMan)
	return &lockMan
}

func (lockman *SInMemoryLockManager) getRecordWithLock(ctx context.Context, key string, new bool) *SInMemoryLockRecord {
	lockman.tableLock.Lock()
	defer lockman.tableLock.Unlock()

	return lockman.getRecord(ctx, key, new)
}

func (lockman *SInMemoryLockManager) getRecord(ctx context.Context, key string, new bool) *SInMemoryLockRecord {
	_, ok := lockman.lockTable[key]
	if !ok {
		if !new {
			return nil
		}
		rec := newInMemoryLockRecord(ctx)
		rec.key = key
		lockman.lockTable[key] = rec
	}
	return lockman.lockTable[key]
}

func (lockman *SInMemoryLockManager) LockKey(ctx context.Context, key string) {
	record := lockman.getRecordWithLock(ctx, key, true)
	record.lockContext(ctx)
}

func (lockman *SInMemoryLockManager) UnlockKey(ctx context.Context, key string) {
	record := lockman.getRecordWithLock(ctx, key, false)
	if record == nil {
		log.Errorf("BUG: unlock an non-existent lock ctx: %p key: %s\n%s", ctx, key, debug.Stack())
		return
	}

	record.unlockContext(ctx)
}
