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

type AccessGroupListOptions struct {
	options.BaseListOptions
}

func (opts *AccessGroupListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type AccessGroupIdOption struct {
	ID string `help:"Access group Id"`
}

func (opts *AccessGroupIdOption) GetId() string {
	return opts.ID
}

func (opts *AccessGroupIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type AccessGroupCreateOptions struct {
	options.BaseCreateOptions
	CloudregionId string
	ManagerId     string
}

func (opts *AccessGroupCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
