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
	"crypto/tls"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"google.golang.org/grpc"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/atexit"
)

type SEtcdLockRecord struct {
	m     *sync.Mutex
	depth int

	pfx  string
	sess *concurrency.Session
	em   *concurrency.Mutex
}

func newEtcdLockRecord(ctx context.Context, lockman *SEtcdLockManager, key string) *SEtcdLockRecord {
	pfx := fmt.Sprintf("%s/%s", lockman.config.LockPrefix, key)
	sess, err := concurrency.NewSession(lockman.cli, concurrency.WithTTL(lockman.config.LockTTL))
	if err != nil {
		panic(fmt.Sprintf("%s: create etcd session: %v", pfx, err))
	}
	em := concurrency.NewMutex(sess, pfx)
	rec := SEtcdLockRecord{
		m:     &sync.Mutex{},
		depth: 0,

		pfx:  pfx,
		sess: sess,
		em:   em,
	}
	return &rec
}

func (rec *SEtcdLockRecord) lockContext(ctx context.Context) {
	rec.m.Lock()
	defer rec.m.Unlock()

	rec.depth += 1
	if rec.depth > 32 {
		// NOTE callers are responsible for ensuring unlock got called
		bug("%s: depth > 32", rec.pfx)
		panic(debug.Stack())
	}

	if err := rec.em.Lock(ctx); err != nil {
		msg := fmt.Sprintf("%s: etcd lock: %v", rec.pfx, err)
		panic(msg)
	}

	if debug_log {
		log.Infof("%s: lock depth %d\n%s", rec.pfx, rec.depth, debug.Stack())
	}
}

func (rec *SEtcdLockRecord) unlockContext(ctx context.Context) (needClean bool) {
	rec.m.Lock()
	defer rec.m.Unlock()

	if debug_log {
		log.Infof("%s: unlock depth %d\n%s", rec.pfx, rec.depth, debug.Stack())
	}

	rec.depth -= 1
	if rec.depth <= 0 {
		if rec.depth < 0 {
			bug("%s: overly unlocked", rec.pfx)
		}

		// Other players can make progress once lock record got removed
		// after this
		//
		// There is no need to unlock rec.em.  Revoke the session lease
		// will trigger etcd to do that.  Should the remote revoke call
		// fail, we expect the lease auto-refresh to stop and the lease
		// will expire in known time limit.
		closed := false
		for i := 0; i < 3; i++ {
			if err := rec.sess.Close(); err != nil {
				log.Errorf("%s: session close: %s", rec.pfx, err)
				if i < 2 {
					time.Sleep(time.Second)
				}
				continue
			}
			closed = true
			break
		}
		if !closed {
			log.Errorf("%s: session close failure", rec.pfx)
		}
		return true
	}
	return false
}

type SEtcdLockManagerConfig struct {
	Endpoints []string
	Username  string
	Password  string
	TLS       *tls.Config

	LockTTL    int
	LockPrefix string

	dialOptions []grpc.DialOption
	dialTimeout time.Duration
}

func (config *SEtcdLockManagerConfig) validate() error {
	if len(config.Endpoints) == 0 {
		return fmt.Errorf("no etcd endpoint configured")
	}
	if config.dialTimeout <= 0 {
		config.dialTimeout = 3 * time.Second
	}
	if len(config.dialOptions) == 0 {
		// let it fail right away
		config.dialOptions = []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTimeout(500 * time.Millisecond),
		}
	}
	if config.LockTTL <= 0 {
		config.LockTTL = 10
	}
	config.LockPrefix = strings.TrimSpace(config.LockPrefix)
	config.LockPrefix = strings.TrimRight(config.LockPrefix, "/")
	if config.LockPrefix == "" {
		return fmt.Errorf("empty etcd lock prefix")
	}

	return nil
}

type SLockTableIndex struct {
	key    string
	holder context.Context
}

type SEtcdLockManager struct {
	*SBaseLockManager
	tableLock *sync.Mutex
	lockTable map[SLockTableIndex]*SEtcdLockRecord

	config *SEtcdLockManagerConfig
	cli    *clientv3.Client
}

func NewEtcdLockManager(config *SEtcdLockManagerConfig) (ILockManager, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: config.Endpoints,
		Username:  config.Username,
		Password:  config.Password,
		TLS:       config.TLS,

		DialOptions: config.dialOptions,
		DialTimeout: config.dialTimeout,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new etcd client")
	}
	lockman := SEtcdLockManager{
		tableLock: &sync.Mutex{},
		lockTable: map[SLockTableIndex]*SEtcdLockRecord{},

		cli:    cli,
		config: config,
	}
	atexit.Register(atexit.ExitHandler{
		Prio:   atexit.PRIO_LOG_OTHER,
		Reason: "etcd-lockman",
		Value:  lockman,
		Func:   atexit.ExitHandlerFunc(lockman.destroyAtExit),
	})
	lockman.SBaseLockManager = NewBaseLockManger(&lockman)
	return &lockman, nil
}

func (lockman *SEtcdLockManager) destroyAtExit(eh atexit.ExitHandler) {
	log.Infof("closing etcd lockman session")
	for _, rec := range lockman.lockTable {
		if err := rec.sess.Close(); err != nil {
			log.Errorf("%s: session close: %v", rec.pfx, err)
		}
	}
	if err := lockman.cli.Close(); err != nil {
		log.Errorf("etcd lockman client close: %v", err)
	}
}

func (lockman *SEtcdLockManager) getRecordWithLock(ctx context.Context, key string) *SEtcdLockRecord {
	lockman.tableLock.Lock()
	defer lockman.tableLock.Unlock()

	return lockman.getRecord(ctx, key, true)
}

func (lockman *SEtcdLockManager) getRecord(ctx context.Context, key string, alloc bool) *SEtcdLockRecord {
	idx := SLockTableIndex{
		key:    key,
		holder: ctx,
	}
	_, ok := lockman.lockTable[idx]
	if !ok {
		if !alloc {
			return nil
		}
		lockman.lockTable[idx] = newEtcdLockRecord(ctx, lockman, key)
	}
	return lockman.lockTable[idx]
}

func (lockman *SEtcdLockManager) LockKey(ctx context.Context, key string) {
	record := lockman.getRecordWithLock(ctx, key)

	record.lockContext(ctx)
}

func (lockman *SEtcdLockManager) UnlockKey(ctx context.Context, key string) {
	lockman.tableLock.Lock()
	defer lockman.tableLock.Unlock()

	record := lockman.getRecord(ctx, key, false)
	if record == nil {
		bug("%s: unlock a non-existent lock\n%s", key, debug.Stack())
		return
	}

	needClean := record.unlockContext(ctx)
	if needClean {
		idx := SLockTableIndex{
			key:    key,
			holder: ctx,
		}
		delete(lockman.lockTable, idx)
	}
}
