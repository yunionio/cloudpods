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

import common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"

type CloudeventOptions struct {
	common_options.CommonOptions
	common_options.DBOptions

	CloudproviderSyncIntervalMinutes int `help:"frequency to sync region cloudprovider task" default:"15"`
	CloudeventSyncIntervalHours      int `help:"frequency to sync cloud event task" default:"1"`

	DisableSyncCloudEvent bool `help:"disable sync cloudevent" default:"true"`

	SyncWithReadEvent bool `help:"sync read operation events" default:"false"`
	OneSyncForHours   int  `help:"Onece sync for hours" default:"1"`
}

var (
	Options CloudeventOptions
)
