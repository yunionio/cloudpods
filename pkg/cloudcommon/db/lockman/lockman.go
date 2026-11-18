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
	"fmt"
	"runtime/debug"
	"sync"

	"yunion.io/x/log"
)

type ILockedClass interface {
	Keyword() string
}

type ILockedObject interface {
	ILockedClass
	GetId() string
}

type ILockManager interface {
	LockKey(ctx context.Context, key string)
	UnlockKey(ctx context.Context, key string)

	LockClass(ctx context.Context, manager ILockedClass, projectId string)
	ReleaseClass(ctx context.Context, manager ILockedClass, projectId string)
	LockObject(ctx context.Context, model ILockedObject)
	ReleaseObject(ctx context.Context, model ILockedObject)
	LockRawObject(ctx context.Context, resName string, resId string)
	ReleaseRawObject(ctx context.Context, resName string, resId string)
	LockJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject)
	ReleaseJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject)
}

func getClassKey(manager ILockedClass, projectId string) string {
	// assert(getattr(cls, '_resource_name_', None) is not None)
	// return '%s-%s' % (cls._resource_name_, user_cred.tenant_id)
	return fmt.Sprintf("%s-%s", manager.Keyword(), projectId)
}

func getObjectKey(model ILockedObject) string {
	// assert(getattr(obj, '_resource_name_', None) is not None)
	// assert(getattr(obj, 'id', None) is not None)
	// return '%s-%s' % (obj._resource_name_, obj.id)
	return getRawObjectKey(model.Keyword(), model.GetId())
}

func getRawObjectKey(resName string, resId string) string {
	return fmt.Sprintf("%s-%s", resName, resId)
}

func getJointObjectKey(model ILockedObject, model2 ILockedObject) string {
	// def _get_joint_object_key(self, obj1, obj2, user_cred):
	// return '%s-%s' % (self._get_object_key(obj1, user_cred),
	//		self._get_object_key(obj2, user_cred))
	return fmt.Sprintf("%s-%s", getObjectKey(model), getObjectKey(model2))
}

var _lockman ILockManager

func Init(man ILockManager) {
	_lockman = man
}

// backgroundCtxWarned remembers the locking paths that were already
// reported, keyed by the offending context kind and the stack of the
// goroutine taking the lock.
var backgroundCtxWarned sync.Map

func checkContext(ctx context.Context) {
	kind := ""
	if ctx == context.Background() {
		kind = "Background"
	} else if ctx == context.TODO() {
		kind = "TODO"
	} else {
		return
	}
	// internal workers (taskman tasks, tickers) legitimately lock objects
	// without an incoming request context; dumping every goroutine of the
	// process on each such lock turned this into a production log flood
	// (thousands of full dumps per process). Report each distinct locking
	// path once, with the current goroutine stack only.
	stack := debug.Stack()
	if _, seen := backgroundCtxWarned.LoadOrStore(kind+"\n"+string(stack), true); !seen {
		log.Warningf("[!!!WARNING!!!] lock context of %s!\n%s", kind, stack)
	}
}

func LockClass(ctx context.Context, manager ILockedClass, projectId string) {
	checkContext(ctx)
	_lockman.LockClass(ctx, manager, projectId)
}

func ReleaseClass(ctx context.Context, manager ILockedClass, projectId string) {
	checkContext(ctx)
	_lockman.ReleaseClass(ctx, manager, projectId)
}

func LockObject(ctx context.Context, model ILockedObject) {
	checkContext(ctx)
	_lockman.LockObject(ctx, model)
}

func ReleaseObject(ctx context.Context, model ILockedObject) {
	checkContext(ctx)
	_lockman.ReleaseObject(ctx, model)
}

func LockRawObject(ctx context.Context, resName string, resId string) {
	checkContext(ctx)
	_lockman.LockRawObject(ctx, resName, resId)
}

func ReleaseRawObject(ctx context.Context, resName string, resId string) {
	checkContext(ctx)
	_lockman.ReleaseRawObject(ctx, resName, resId)
}

func LockJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	checkContext(ctx)
	_lockman.LockJointObject(ctx, model, model2)
}

func ReleaseJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	checkContext(ctx)
	_lockman.ReleaseJointObject(ctx, model, model2)
}
