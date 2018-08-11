package candidate

import (
	"flag"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/scheduler/cache"
	"yunion.io/x/onecloud/pkg/scheduler/cache/db"
	"yunion.io/x/onecloud/pkg/scheduler/cache/sync"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
)

var (
	// flag to connect database
	dialect = flag.String("db-dialect", "mysql", "db dialect")
	dbURL   = flag.String("db-url", "root:root@tcp(127.0.0.1:3306)/mclouds?charset=utf8&parseTime=True", "db url")

	// Kinds of cache manager
	testDBMan   *cache.GroupManager
	testSyncMan *cache.GroupManager
	//testCandiMan *cache.GroupManager
)

func init() {
	if err := models.Init(*dialect, *dbURL); err != nil {
		panic(err)
	}

	stopCh := make(chan struct{})
	testDBMan = db.NewCacheManager(stopCh)
	testDBMan.Run()
	testSyncMan = sync.NewSyncManager(stopCh)
	testSyncMan.Run()
	//testCandiMan = NewCandidateManager(testCacheMan, testSyncMan, stopCh)
	//testCandiMan.Run()
}

func TestHostBuildOne(t *testing.T) {
	time.Sleep(3 * time.Second)
	builder := &HostBuilder{}
	err := builder.init([]string{"01ee5aca-3d63-404b-957c-fb9ea2306770"}, testDBMan, testSyncMan)
	if err != nil {
		t.Fatal(err)
	}
	descs, err := builder.buildOne(builder.hosts[0].(*models.Host))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(descs)
}

func BenchmarkParallelizeBuild(b *testing.B) {
	time.Sleep(2 * time.Second)
	builder := &HostBuilder{}
	ids, err := models.AllIDs(models.Hosts)
	if err != nil {
		b.Fatal(err)
	}
	err = builder.init(ids, testDBMan, testSyncMan)
	if err != nil {
		b.Fatal(err)
	}
	for n := 0; n < b.N; n++ {
		for _, h := range builder.hosts {
			_, err = builder.buildOne(h.(*models.Host))
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
