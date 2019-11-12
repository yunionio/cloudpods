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
	"sync"
	"testing"

	"yunion.io/x/onecloud/pkg/util/atexit"
)

func TestEctdLockManager(t *testing.T) {
	cfgs := []*testLockManagerConfig{}
	shared := newSharedObject()
	for i := 0; i < 4; i++ {
		lockman, err := NewEtcdLockManager(&SEtcdLockManagerConfig{
			Endpoints: []string{"localhost:2379"},
		})
		if err != nil {
			t.Skipf("new etcd lockman: %v", err)
		}
		cfg := &testLockManagerConfig{
			players: 3,
			cycles:  3,
			lockman: lockman,
			shared:  shared,
		}
		cfgs = append(cfgs, cfg)
	}
	defer atexit.Handle()
	wg := &sync.WaitGroup{}
	wg.Add(len(cfgs))
	for _, cfg := range cfgs {
		go func(cfg *testLockManagerConfig) {
			testLockManager(t, cfg)
			wg.Done()
		}(cfg)
	}
	wg.Wait()
}
