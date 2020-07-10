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

	"yunion.io/x/log"
)

type SNoopLockManager struct {
	*SBaseLockManager
}

func (lockman *SNoopLockManager) LockKey(ctx context.Context, key string) {
	log.Debugf("LockKey %s in context %#v", key, ctx)
}

func (lockman *SNoopLockManager) UnlockKey(ctx context.Context, key string) {
	log.Debugf("UnlockKey %s in context %#v", key, ctx)
}

func NewNoopLockManager() ILockManager {
	lockMan := SNoopLockManager{}
	lockMan.SBaseLockManager = NewBaseLockManger(&lockMan)
	return &lockMan
}
