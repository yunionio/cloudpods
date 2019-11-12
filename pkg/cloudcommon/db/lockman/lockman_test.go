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
	"sync"
	"testing"
	"time"

	"yunion.io/x/pkg/util/stringutils"
)

type FakeObject struct {
	Id string

	playerId      int
	playerIdCount int
}

func (o *FakeObject) GetId() string {
	return o.Id
}

func (o *FakeObject) Keyword() string {
	return "fake"
}

func (o *FakeObject) push(playerId int) {
	if o.playerId >= 0 {
		if o.playerId != playerId {
			panic(fmt.Sprintf("obj locked by %d, locked again by %d", o.playerId, playerId))
		}
	} else {
		if o.playerIdCount != 0 {
			panic(fmt.Sprintf("obj unlocked but player id count %d", o.playerIdCount))
		}
		o.playerId = playerId
	}
	o.playerIdCount += 1
}

func (o *FakeObject) pop(playerId int) {
	if o.playerId != playerId {
		panic(fmt.Sprintf("obj previously locked by %d, now unlocked by %d", o.playerId, playerId))
	}
	o.playerIdCount -= 1
	if o.playerIdCount < 0 {
		panic(fmt.Sprintf("obj overly unlocked"))
	} else if o.playerIdCount == 0 {
		o.playerId = -1
	}
}

func newSharedObject() *FakeObject {
	obj := &FakeObject{
		Id: stringutils.UUID4(),

		playerId:      -1,
		playerIdCount: 0,
	}
	return obj
}

type testLockManagerConfig struct {
	players int
	cycles  int
	shared  *FakeObject
	lockman ILockManager
}

func testLockManager(t *testing.T, cfg *testLockManagerConfig) {
	players := cfg.players
	cycles := cfg.cycles
	shared := cfg.shared
	lockman := cfg.lockman

	bug = func(fmtStr string, fmtArgs ...interface{}) {
		t.Fatalf(fmtStr, fmtArgs...)
	}
	run := func(ctx context.Context, id int, cycle int, sleep time.Duration) {
		key := getObjectKey(shared)
		indents := []string{"  ", "    "}
		logpref := fmt.Sprintf("%p: %02d: %p", lockman, id, ctx)
		func() {
			fmt.Printf("%s: %s >acquire\n", logpref, indents[0])
			lockman.LockKey(ctx, key)
			shared.push(id)
			fmt.Printf("%s: %s >>acquired\n", logpref, indents[1])
		}()
		defer func() {
			fmt.Printf("%s: %s <<release\n", logpref, indents[1])
			shared.pop(id)
			lockman.UnlockKey(ctx, key)
			fmt.Printf("%s: %s <released\n", logpref, indents[0])
		}()
		time.Sleep(sleep)
	}

	var wg sync.WaitGroup
	for id := 0; id < players; id += 1 {
		wg.Add(1)
		go func(localId int) {
			ctx := context.WithValue(context.Background(), "ID", localId)
			for i := 0; i < cycles; i += 1 {
				run(ctx, localId, i, time.Duration(localId)*time.Second)
			}
			wg.Done()
		}(id)
	}

	wg.Wait()
}
