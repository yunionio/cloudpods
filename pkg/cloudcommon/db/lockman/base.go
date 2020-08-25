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

import "context"

type SBaseLockManager struct {
	manager ILockManager
}

func NewBaseLockManger(m ILockManager) *SBaseLockManager {
	return &SBaseLockManager{manager: m}
}

func (m *SBaseLockManager) LockClass(ctx context.Context, manager ILockedClass, projectId string) {
	key := getClassKey(manager, projectId)
	m.manager.LockKey(ctx, key)
}

func (m *SBaseLockManager) ReleaseClass(ctx context.Context, manager ILockedClass, projectId string) {
	key := getClassKey(manager, projectId)
	m.manager.UnlockKey(ctx, key)
}

func (m *SBaseLockManager) LockObject(ctx context.Context, model ILockedObject) {
	key := getObjectKey(model)
	m.manager.LockKey(ctx, key)
}

func (m *SBaseLockManager) ReleaseObject(ctx context.Context, model ILockedObject) {
	key := getObjectKey(model)
	m.manager.UnlockKey(ctx, key)
}

func (m *SBaseLockManager) LockRawObject(ctx context.Context, resName string, resId string) {
	key := getRawObjectKey(resName, resId)
	m.manager.LockKey(ctx, key)
}

func (m *SBaseLockManager) ReleaseRawObject(ctx context.Context, resName string, resId string) {
	key := getRawObjectKey(resName, resId)
	m.manager.UnlockKey(ctx, key)
}

func (m *SBaseLockManager) LockJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	key := getJointObjectKey(model, model2)
	m.manager.LockKey(ctx, key)
}

func (m *SBaseLockManager) ReleaseJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	key := getJointObjectKey(model, model2)
	m.manager.UnlockKey(ctx, key)
}
