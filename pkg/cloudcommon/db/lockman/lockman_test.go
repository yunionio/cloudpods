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
	"sync/atomic"
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

func testLockManager(t *testing.T) {
	objId := stringutils.UUID4()
	players := 3
	cycles := 3
	bits := uint32(0)

	bug = func(fmtStr string, fmtArgs ...interface{}) {
		t.Fatalf(fmtStr, fmtArgs...)
	}
	run := func(ctx context.Context, obj ILockedObject, id int, cycle int, sleep time.Duration) {
		the_val := uint32(1) << uint(id)
		indents := []string{"  ", "    "}
		func() {
			fmt.Printf("%02d: %p: %s >acquire\n", id, ctx, indents[0])
			LockObject(ctx, obj)
			v := atomic.LoadUint32(&bits)
			if v != 0 {
				t.Fatalf("lock already taken: %x", v)
			}
			if !atomic.CompareAndSwapUint32(&bits, 0, the_val) {
				t.Fatalf("lock stolen 0: %x", bits)
			}
			fmt.Printf("%02d: %p: %s >>acquired\n", id, ctx, indents[1])
		}()
		defer func() {
			fmt.Printf("%02d: %p: %s <<release\n", id, ctx, indents[1])
			v := atomic.LoadUint32(&bits)
			if v != the_val {
				t.Fatalf("lock stolen 1: %x", v)
			}
			if !atomic.CompareAndSwapUint32(&bits, the_val, 0) {
				t.Fatalf("lock stolen 2: %x", v)
			}
			ReleaseObject(ctx, obj)
			fmt.Printf("%02d: %p: %s <released\n", id, ctx, indents[0])
		}()
		time.Sleep(sleep)
	}

	var wg sync.WaitGroup
	for id := 0; id < players; id += 1 {
		wg.Add(1)
		go func(localId int) {
			ctx := context.WithValue(context.Background(), "ID", localId)
			for i := 0; i < cycles; i += 1 {
				obj := &FakeObject{Id: objId}
				run(ctx, obj, localId, i, time.Duration(localId)*time.Second)
			}
			wg.Done()
		}(id)
	}

	wg.Wait()
}
