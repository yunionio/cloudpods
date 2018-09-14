package main

import (
	"context"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"

	"math/rand"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
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

func run(ctx context.Context, obj lockman.ILockedObject, id int, sleep time.Duration) {
	log.Infof("ready to run at %d [%p]", id, ctx)
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)
	log.Infof("Acquire obj at %d [%p]", id, ctx)
	time.Sleep(sleep)
	log.Infof("Release obj at %d [%p]", id, ctx)
}

func main() {
	lockman.Init(lockman.NewInMemoryLockManager())
	objId := stringutils.UUID4()
	cycle := 10

	var wg sync.WaitGroup

	log.Infof("Start")

	for id := 0; id <= 3; id += 1 {
		wg.Add(1)
		go func(localId int) {
			log.Infof("Start %d", localId)
			ctx := context.WithValue(context.Background(), "ID", localId)
			for i := 0; i < cycle; i += 1 {
				obj := &FakeObject{Id: objId}
				run(ctx, obj, localId, time.Duration(rand.Intn(1000))*time.Millisecond)
			}
			wg.Done()
		}(id)
	}

	wg.Wait()
}
