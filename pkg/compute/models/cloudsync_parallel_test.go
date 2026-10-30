// Copyright 2019 Yunion
package models

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/compute/options"
)

func TestSSyncResultSetConcurrentAdd(t *testing.T) {
	set := NewSSyncResultSet()
	const n = 100
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func() {
			unlock := set.lock()
			res := set.getOrCreate("tests")
			res.AddCnt++
			unlock()
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	unlock := set.lock()
	res := set.data["tests"]
	unlock()
	if res == nil || res.AddCnt != n {
		t.Fatalf("expected AddCnt=%d got %#v", n, res)
	}
}

func TestRunSyncTasksParallelCancel(t *testing.T) {
	old := options.Options.CloudRegionResourceSyncConcurrency
	options.Options.CloudRegionResourceSyncConcurrency = 3
	defer func() { options.Options.CloudRegionResourceSyncConcurrency = old }()

	ctx, cancel := context.WithCancel(context.Background())
	var ran int32
	tasks := make([]func(), 0, 10)
	for i := 0; i < 10; i++ {
		tasks = append(tasks, func() {
			atomic.AddInt32(&ran, 1)
			time.Sleep(50 * time.Millisecond)
		})
	}
	cancel()
	runSyncTasksParallel(ctx, tasks)
	if atomic.LoadInt32(&ran) > 10 {
		t.Fatalf("unexpected ran=%d", ran)
	}
}

func TestRunSyncTasksParallelLimit(t *testing.T) {
	old := options.Options.CloudRegionResourceSyncConcurrency
	options.Options.CloudRegionResourceSyncConcurrency = 2
	defer func() { options.Options.CloudRegionResourceSyncConcurrency = old }()

	var peak, cur int32
	tasks := make([]func(), 0, 6)
	for i := 0; i < 6; i++ {
		tasks = append(tasks, func() {
			v := atomic.AddInt32(&cur, 1)
			for {
				p := atomic.LoadInt32(&peak)
				if v <= p || atomic.CompareAndSwapInt32(&peak, p, v) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			atomic.AddInt32(&cur, -1)
		})
	}
	runSyncTasksParallel(context.Background(), tasks)
	if peak > 2 {
		t.Fatalf("peak concurrency %d > limit 2", peak)
	}
	if peak < 1 {
		t.Fatalf("expected some concurrency, peak=%d", peak)
	}
}

// TestStorageCachePairsConcurrentMerge 模拟 Wave3 多 zone 并行时对 storageCachePairs 的快照+合并。
func TestStorageCachePairsConcurrentMerge(t *testing.T) {
	old := options.Options.CloudRegionResourceSyncConcurrency
	options.Options.CloudRegionResourceSyncConcurrency = 3
	defer func() { options.Options.CloudRegionResourceSyncConcurrency = old }()

	var pairsMu sync.Mutex
	storageCachePairs := make([]sStoragecacheSyncPair, 0)
	tasks := make([]func(), 0, 6)
	for i := 0; i < 6; i++ {
		tasks = append(tasks, func() {
			pairsMu.Lock()
			_ = append([]sStoragecacheSyncPair(nil), storageCachePairs...)
			pairsMu.Unlock()
			time.Sleep(5 * time.Millisecond)
			pairsMu.Lock()
			storageCachePairs = append(storageCachePairs, sStoragecacheSyncPair{isNew: true})
			pairsMu.Unlock()
		})
	}
	runSyncTasksParallel(context.Background(), tasks)
	if len(storageCachePairs) != 6 {
		t.Fatalf("expected 6 pairs, got %d", len(storageCachePairs))
	}
}

func TestSSyncResultSetMarshalJSON(t *testing.T) {
	set := NewSSyncResultSet()
	unlock := set.lock()
	res := set.getOrCreate("compute")
	res.AddCnt = 3
	unlock()
	b, err := set.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 10 || string(b) == "{}" {
		t.Fatalf("unexpected marshal %s", b)
	}
}

func TestSyncResultGroupByKeyword(t *testing.T) {
	cases := map[string]string{
		"hosts":                   "compute",
		"disks":                   "compute",
		"hoststorages":            "compute",
		"servers":                 "compute",
		"vpcs":                    "network",
		"networks":                "network",
		"secgroups":               "network",
		"eips":                    "network",
		"buckets":                 "objectstore",
		"loadbalancers":           "loadbalancer",
		"loadbalanceracls":        "loadbalancer",
		"dbinstances":             "rds",
		"dbinstancebackups":       "rds",
		"elasticcaches":           "cache",
		"file_systems":            "nas",
		"access_groups":           "nas",
		"waf_instances":           "waf",
		"cloud_kube_clusters":     "container",
		"cachedimages":            "image",
		"cloudproviderquotas":     "quota",
		"zones":                   "zone",
		"unknown_custom_resource": "unknown_custom_resource",
	}
	for kw, want := range cases {
		if got := syncResultGroupByKeyword(kw); got != want {
			t.Fatalf("%s: got %s want %s", kw, got, want)
		}
	}
}
