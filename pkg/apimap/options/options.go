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

	"yunion.io/x/onecloud/pkg/apihelper"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SOptions struct {
	common_options.CommonOptions

	// APISyncIntervalSeconds is how often the model sets are synced from the
	// region API.  Keep it below the sync interval of the services that
	// consume the model sets, so that a change reaches them without an extra
	// delay.
	APISyncIntervalSeconds int `default:"5" help:"interval in seconds to sync model sets from region"`
	// APIRunDelayMilliseconds is how long a triggered sync waits before it
	// runs, so that a burst of triggers collapses into one sync.
	APIRunDelayMilliseconds int `default:"100" help:"delay in milliseconds before a scheduled sync runs"`
	// APIListBatchSize is the batch size of the model list requests made
	// against the region API.
	APIListBatchSize int `default:"1024" help:"batch size of model list requests"`
}

var (
	opts SOptions
)

func GetOptions() *SOptions {
	return &opts
}

func Init() {
	common_options.ParseOptions(&opts, os.Args, "apimap.conf", "apimap")
}

func (opts *SOptions) ValidateThenInit() error {
	if opts.APISyncIntervalSeconds < apihelper.MinSyncIntervalSeconds {
		opts.APISyncIntervalSeconds = apihelper.MinSyncIntervalSeconds
	}
	if opts.APIRunDelayMilliseconds < apihelper.MinRunDelayMilliseconds {
		opts.APIRunDelayMilliseconds = apihelper.MinRunDelayMilliseconds
	}
	if opts.APIListBatchSize <= 20 {
		opts.APIListBatchSize = 20
	}
	return nil
}

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*SOptions)
	newOpts := newO.(*SOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}

	return changed
}
