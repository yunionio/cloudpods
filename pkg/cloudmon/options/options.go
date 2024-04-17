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
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type CloudMonOptions struct {
	common_options.CommonOptions
	PingProbeOptions

	ResourcesSyncInterval   int64  `help:"Increment Sync Interval unit:minute" default:"10"`
	CollectMetricInterval   int64  `help:"Increment Sync Interval unit:minute" default:"6"`
	SkipMetricPullProviders string `help:"Skip indicate provider metric pull" default:""`

	InfluxDatabase string `help:"influxdb database name, default telegraf" default:"telegraf"`

	DisableServiceMetric               bool  `help:"disable service metric collect"`
	CollectServiceMetricIntervalMinute int64 `help:"Collect Service metirc Interval unit:minute" default:"5"`

	HistoryMetricPullDays          int  `help:"pull history metrics" default:"-1"`
	SupportAzureTableStorageMetric bool `help:"support collect azure memory and disk usage metric, there may be additional charges" default:"false"`

	CloudAccountCollectMetricsBatchCount        int `help:"Cloud Account Collect Metrics Batch Count" default:"10"`
	CloudResourceCollectMetricsBatchCount       int `help:"Cloud Resource Collect Metrics BatchC ount" default:"40"`
	OracleCloudResourceCollectMetricsBatchCount int `help:"OracleCloud Resource Collect Metrics BatchC ount" default:"1"`
}

type PingProbeOptions struct {
	Debug         bool `help:"debug"`
	ProbeCount    int  `help:"probe count, default is 3" default:"3"`
	TimeoutSecond int  `help:"probe timeout in second, default is 1 second" default:"1"`

	DisablePingProbe      bool  `help:"enable ping probe"`
	PingProbIntervalHours int64 `help:"PingProb Interval unit:hour" default:"6"`

	PingReserveIPTimeoutHours int `help:"expire hours to reserve the probed IP, default 0, never expire" default:"0"`
}

var (
	Options CloudMonOptions
)

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*CloudMonOptions)
	newOpts := newO.(*CloudMonOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}

	if oldOpts.DisablePingProbe != newOpts.DisablePingProbe {
		if !oldOpts.IsSlaveNode {
			changed = true
		}
	}
	if oldOpts.ResourcesSyncInterval != newOpts.ResourcesSyncInterval {
		changed = true
	}
	if oldOpts.CollectMetricInterval != newOpts.CollectMetricInterval {
		changed = true
	}
	if oldOpts.CollectServiceMetricIntervalMinute != newOpts.CollectServiceMetricIntervalMinute {
		changed = true
	}

	return changed
}
