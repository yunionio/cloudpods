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
	"math/rand"
	"testing"
	"time"
)

func TestInMemoryLockManager(t *testing.T) {
	lockman := NewInMemoryLockManager()
	shared := newSharedObject()
	cfg := &testLockManagerConfig{
		players: 3,
		cycles:  3,
		lockman: lockman,
		shared:  shared,
	}
	testLockManager(t, cfg)
}

func TestRunManu(t *testing.T) {
	rand.Seed(100)
	lockman := NewInMemoryLockManager()
	counter := 0
	MAX := 512
	for i := 0; i < MAX; i++ {
		go func() {
			ctx := context.Background()
			now := time.Now()
			ms := now.UnixMilli()
			ctx = context.WithValue(ctx, "Time", ms)
			lockman.LockKey(ctx, "test")
			defer lockman.UnlockKey(ctx, "test")
			counter++
			time.Sleep(1 + time.Duration(rand.Intn(50))*time.Millisecond)
			t.Logf("counter: %d", counter)
		}()
	}
	for counter < MAX {
		time.Sleep(1)
	}
	t.Logf("complete")
}
