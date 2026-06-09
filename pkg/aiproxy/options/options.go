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

type SAiProxyOptions struct {
	common_options.CommonOptions
	common_options.DBOptions

	AdvertiseAddress             string `help:"Standby node address advertised to clients, e.g. http://10.0.0.2:30889; default derives from bind address and port" default:""`
	NodeHeartbeatIntervalSeconds int    `help:"Interval in seconds for standby node registration heartbeat" default:"60"`
}

var (
	Options SAiProxyOptions
)

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*SAiProxyOptions)
	newOpts := newO.(*SAiProxyOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	if common_options.OnDBOptionsChange(&oldOpts.DBOptions, &newOpts.DBOptions) {
		changed = true
	}
	return changed
}
