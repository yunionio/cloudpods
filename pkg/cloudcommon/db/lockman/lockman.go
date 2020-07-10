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

func LockClass(ctx context.Context, manager ILockedClass, projectId string) {
	_lockman.LockClass(ctx, manager, projectId)
}

func ReleaseClass(ctx context.Context, manager ILockedClass, projectId string) {
	_lockman.ReleaseClass(ctx, manager, projectId)
}

func LockObject(ctx context.Context, model ILockedObject) {
	_lockman.LockObject(ctx, model)
}

func ReleaseObject(ctx context.Context, model ILockedObject) {
	_lockman.ReleaseObject(ctx, model)
}

func LockRawObject(ctx context.Context, resName string, resId string) {
	_lockman.LockRawObject(ctx, resName, resId)
}

func ReleaseRawObject(ctx context.Context, resName string, resId string) {
	_lockman.ReleaseRawObject(ctx, resName, resId)
}

func LockJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	_lockman.LockJointObject(ctx, model, model2)
}

func ReleaseJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	_lockman.ReleaseJointObject(ctx, model, model2)
}
