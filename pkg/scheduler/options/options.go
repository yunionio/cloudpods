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

	api "yunion.io/x/onecloud/pkg/apis/scheduler"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type SchedulerOptions struct {
	options.ComputeOptions

	SchedOptions

	// gin http framework mode
	// GinMode string `help:"gin http framework work mode" default:"debug" choices:"debug|release"`
}

type SchedOptions struct {
	SchedulerPort           int  `help:"The port that the scheduler's http service runs on" default:"8897"`
	IgnoreFakeDeletedGuests bool `help:"Ignore fake deleted guests when build host memory and cpu size" default:"false"`

	AlwaysCheckAllPredicates    bool   `help:"Excute all predicates when scheduling" default:"false"`
	DisableBaremetalPredicates  bool   `help:"Switch to trigger baremetal related predicates" default:"false"`
	SchedulerTestLimit          int    `help:"Scheduler test items' limitations" default:"100"`
	SchedulerHistoryLimit       int    `help:"Scheduler history items' limitations" default:"1000"`
	SchedulerHistoryCleanPeriod string `help:"Scheduler history cleanup period" default:"60s"`

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

	ContainerNumaAllocate            bool `help:"Allocate numa pin for container guests" default:"false"`
	GuestCpusetAllocSequence         bool `help:"Guest alloc cpuset sequence" default:"false"`
	GuestCpusetAllocSequenceInterval int  `help:"Guest alloc cpuset sequence interval" default:"4"`

	OpenstackOptions
}

type OpenstackOptions struct {
	OpenstackSchedulerCPUFilter     bool `help:"Scheduler OpenStack usable host by cpu" default:"true"`
	OpenstackSchedulerMemoryFilter  bool `help:"Scheduler OpenStack usable host by memory" default:"true"`
	OpenstackSchedulerStorageFilter bool `help:"Scheduler OpenStack usable host by storage" default:"true"`
	OpenstackSchedulerSKUFilter     bool `help:"Scheduler OpenStack usable host by sku" default:"false"`
}

func OnOpenstackOptionsChange(oOpts, nOpts interface{}) bool {
	oldOpts := oOpts.(*OpenstackOptions)
	newOpts := nOpts.(*OpenstackOptions)

	if oldOpts.OpenstackSchedulerCPUFilter != newOpts.OpenstackSchedulerCPUFilter {
		return true
	}
	if oldOpts.OpenstackSchedulerMemoryFilter != newOpts.OpenstackSchedulerMemoryFilter {
		return true
	}
	if oldOpts.OpenstackSchedulerStorageFilter != newOpts.OpenstackSchedulerStorageFilter {
		return true
	}
	if oldOpts.OpenstackSchedulerSKUFilter != newOpts.OpenstackSchedulerSKUFilter {
		return true
	}
	return false
}

var (
	Options SchedulerOptions
)

func Init() {
	common_options.ParseOptions(&Options, os.Args, "region.conf", api.SERVICE_TYPE)
	options.Options = Options.ComputeOptions
}

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*SchedulerOptions)
	newOpts := newO.(*SchedulerOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	if common_options.OnDBOptionsChange(&oldOpts.DBOptions, &newOpts.DBOptions) {
		changed = true
	}
	if OnOpenstackOptionsChange(&oldOpts.OpenstackOptions, &newOpts.OpenstackOptions) {
		changed = true
	}

	options.Options = newOpts.ComputeOptions

	return changed
}
