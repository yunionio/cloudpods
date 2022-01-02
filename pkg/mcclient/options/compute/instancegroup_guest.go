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

package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type InstanceGroupGuestListOptions struct {
	options.BaseListOptions

	Group  string `help:"Group ID or Name"`
	Server string `help:"Gest ID or Name"`
}

func (opts *InstanceGroupGuestListOptions) GetMasterOpt() string {
	return opts.Group
}

func (opts *InstanceGroupGuestListOptions) GetSlaveOpt() string {
	return opts.Server
}

func (opts *InstanceGroupGuestListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if opts.Group != "" {
		params.Add(jsonutils.NewString(opts.Group), "group_id")
	}
	if opts.Server != "" {
		params.Add(jsonutils.NewString(opts.Server), "guest_id")
	}
	return params, nil
}
