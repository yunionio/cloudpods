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

type GlobalVpcListOptions struct {
	options.BaseListOptions

	OrderByVpcCount string
}

func (opts *GlobalVpcListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type GlobalVpcIdOption struct {
	ID string `help:"Global vpc Id"`
}

func (opts *GlobalVpcIdOption) GetId() string {
	return opts.ID
}

func (opts *GlobalVpcIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type GlobalVpcCreateOptions struct {
	NAME    string `help:"Global vpc name"`
	MANAGER string `help:"Cloudprovider Id"`
}

func (opts *GlobalVpcCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}
