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
	"sync"
	"testing"
	"time"

	"yunion.io/x/pkg/util/stringutils"
)

type FakeObject struct {
	Id string
}

func (o *FakeObject) GetId() string {
	return o.Id
}

func (o *FakeObject) Keyword() string {
	return "fake"
}

func run(t *testing.T, ctx context.Context, obj ILockedObject, id int, sleep time.Duration) {
	t.Logf("ready to run at %d [%p]", id, ctx)
	LockObject(ctx, obj)
	defer ReleaseObject(ctx, obj)
	t.Logf("Acquire obj at %d [%p]", id, ctx)
	time.Sleep(sleep)
	t.Logf("Release obj at %d [%p]", id, ctx)
}

func TestInMemoryLockManager(t *testing.T) {
	Init(NewInMemoryLockManager())
	objId := stringutils.UUID4()
	cycle := 1

	var wg sync.WaitGroup

	t.Log("Start")

	for id := 0; id <= 3; id += 1 {
		wg.Add(1)
		go func(localId int) {
			t.Logf("Start %d", localId)
			ctx := context.WithValue(context.Background(), "ID", localId)
			for i := 0; i < cycle; i += 1 {
				obj := &FakeObject{Id: objId}
				run(t, ctx, obj, localId, time.Duration(localId)*time.Second)
			}
			wg.Done()
		}(id)
	}

	wg.Wait()
}
