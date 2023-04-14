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

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type SOptions struct {
	options.ComputeOptions
}

var (
	opts SOptions
)

func GetOptions() *SOptions {
	return &opts
}

func Init() {
	common_options.ParseOptions(&opts, os.Args, "apimap.conf", "apimap")
	options.Options = opts.ComputeOptions
}
func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*SOptions)
	newOpts := newO.(*SOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	if common_options.OnDBOptionsChange(&oldOpts.DBOptions, &newOpts.DBOptions) {
		changed = true
	}

	options.Options = newOpts.ComputeOptions

	return changed
}
