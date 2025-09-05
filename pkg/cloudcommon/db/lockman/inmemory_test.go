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
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/petermattis/goid"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
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
			ctx = context.WithValue(ctx, "time", ms)
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

type app struct {
	ctx     context.Context
	key     string
	lockman ILockManager
}

func (app *app) run() {
	app.lockman.LockKey(app.ctx, app.key)
	defer app.lockman.UnlockKey(app.ctx, app.key)

	fmt.Printf("run for goid: %d key %s\n", goid.Get(), app.key)
}

type emptyKey struct{}

func TestRunManu3(t *testing.T) {
	for i := 0; i < 100; i++ {
		TestRunManu2(t)
	}
}

func TestRunManu2(t *testing.T) {
	rand.Seed(100)
	lockman := NewInMemoryLockManager()

	debug_log = true

	// 使用 WaitGroup 来等待所有 goroutine 完成
	var wg sync.WaitGroup
	MAX_GOROUTINE := 4096
	MAX_KEY := 4

	complete := make(chan struct{})

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				t.Logf("timeout")
				utils.DumpAllGoroutineStack(os.Stdout)
			case <-complete:
				return
			}
		}
	}()

	bgCtx := context.Background()

	// 为每个 goroutine 创建独立的 context
	for i := 0; i < MAX_GOROUTINE; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 为每个 goroutine 创建新的 context
			ctx := context.WithValue(bgCtx, emptyKey{}, id)
			log.Infof("ctx for id %d: %p", id, ctx)
			app := &app{
				ctx:     ctx,
				key:     fmt.Sprintf("test-%d", id%MAX_KEY),
				lockman: lockman,
			}
			app.run()
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	close(complete)
	t.Logf("complete")
}
