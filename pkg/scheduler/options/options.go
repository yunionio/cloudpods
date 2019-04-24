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

package options

import (
	"os"

	common_optoins "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type SchedulerOptions struct {
	options.ComputeOptions

	SchedOptions

	// gin http framework mode
	GinMode string `help:"gin http framework work mode" default:"debug" choices:"debug|release"`
}

type SchedOptions struct {
	SchedulerPort           int  `help:"The port that the scheduler's http service runs on" default:"8897"`
	IgnoreFakeDeletedGuests bool `help:"Ignore fake deleted guests when build host memory and cpu size" default:"false"`

	AlwaysCheckAllPredicates    bool   `help:"Excute all predicates when scheduling" default:"false"`
	DisableBaremetalPredicates  bool   `help:"Switch to trigger baremetal related predicates" default:"false"`
	SchedulerTestLimit          int    `help:"Scheduler test items' limitations" default:"100"`
	SchedulerHistoryLimit       int    `help:"Scheduler history items' limitations" default:"1000"`
	SchedulerHistoryCleanPeriod string `help:"Scheduler history cleanup period" default:"60s"`

	// per isolated device default reserverd resource
	MemoryReservedPerIsolatedDevice  int64 `help:"Per isolated device default reserverd memory size in MB" default:"8192"`    // 8G
	CpuReservedPerIsolatedDevice     int64 `help:"Per isolated device default reserverd CPU count" default:"8"`               // 8 core
	StorageReservedPerIsolatedDevice int64 `help:"Per isolated device default reserverd storage size in MB" default:"102400"` // 100G

	// parallelization options
	HostBuildParallelizeSize int `help:"Number of host description build parallelization" default:"14"`
	PredicateParallelizeSize int `help:"Number of execute predicates parallelization" default:"14"`
	PriorityParallelizeSize  int `help:"Number of execute priority parallelization" default:"14"`

	// expire queue options
	ExpireQueueConsumptionPeriod  string `help:"Expire queue consumption period" default:"3s"`
	ExpireQueueConsumptionTimeout string `help:"Expire queue consumption timeout" default:"10s"`
	ExpireQueueMaxLength          int    `help:"Expire queue max length" default:"1000"`
	ExpireQueueDealLength         int    `help:"Expire queue deal length" default:"100"`

	// completed queue options
	CompletedQueueConsumptionPeriod  string `help:"Completed queue consumption period" default:"30s"`
	CompletedQueueConsumptionTimeout string `help:"Completed queue consumption timeout" default:"30s"`
	CompletedQueueMaxLength          int    `help:"Completed queue max length" default:"100"`
	CompletedQueueDealLength         int    `help:"Completed queue deal length" default:"10"`

	// cache options
	HostCandidateCacheTTL         string `help:"Build host description candidate cache TTL" default:"0s"`
	HostCandidateCacheReloadCount int    `help:"Build host description candidate cache reload times count" default:"20"`
	HostCandidateCachePeriod      string `help:"Build host description candidate cache period" default:"30s"`

	BaremetalCandidateCacheTTL         string `help:"Build Baremetal description candidate cache TTL" default:"0s"`
	BaremetalCandidateCacheReloadCount int    `help:"Build Baremetal description candidate cache reload times count" default:"20"`
	BaremetalCandidateCachePeriod      string `help:"Build Baremetal description candidate cache period" default:"30s"`

	NetworkCacheTTL    string `help:"Build network info from database to cache TTL" default:"0s"`
	NetworkCachePeriod string `help:"Build network info from database to cache TTL" default:"1m"`

	ClusterDBCacheTTL    string `help:"Cluster database cache TTL" default:"0s"`
	ClusterDBCachePeriod string `help:"Cluster database cache period" default:"5m"`

	BaremetalAgentDBCacheTTL    string `help:"BaremetalAgent database cache TTL" default:"0s"`
	BaremetalAgentDBCachePeriod string `help:"BaremetalAgent database cache period" default:"5m"`

	AggregateDBCacheTTL    string `help:"Aggregate database cache TTL" default:"0s"`
	AggregateDBCachePeriod string `help:"Aggregate database cache period" default:"30s"`

	AggregateHostDBCacheTTL    string `help:"AggregateHost database cache TTL" default:"0s"`
	AggregateHostDBCachePeriod string `help:"AggregateHost database cache period" default:"30s"`

	NetworksDBCacheTTL    string `help:"Networks database cache TTL" default:"0s"`
	NetworksDBCachePeriod string `help:"Networks database cache period" default:"5m"`

	NetinterfaceDBCacheTTL    string `help:"Netinterfaces database cache TTL" default:"0s"`
	NetinterfaceDBCachePeriod string `help:"Netinterfaces database cache period" default:"5m"`

	WireDBCacheTTL    string `help:"Wire database cache TTL" default:"0s"`
	WireDBCachePeriod string `help:"Wire database cache period" default:"5m"`

	SkuRefreshInterval string `help:"Server SKU refresh interval" default:"12h"`
}

var (
	opt SchedulerOptions
)

func GetOptions() *SchedulerOptions {
	return &opt
}

func Init() {
	common_optoins.ParseOptions(&opt, os.Args, "region.conf", "scheduler")
	options.Options = opt.ComputeOptions
}
