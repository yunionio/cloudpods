package lockman

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/yunionio/pkg/util/stringutils"
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
	defer LockObject(ctx, obj)
	t.Logf("Acquire obj at %s [%p]", id, ctx)
	time.Sleep(sleep)
	t.Logf("Release obj at %s [%p]", id, ctx)
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
				run(t, ctx, obj, localId, time.Duration(id)*time.Second)
			}
			wg.Done()
		}(id)
	}

	wg.Wait()
}
