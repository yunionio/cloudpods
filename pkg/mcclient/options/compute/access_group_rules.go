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

type AccessGroupRuleListOptions struct {
	options.BaseListOptions
}

func (opts *AccessGroupRuleListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type AccessGroupRuleIdOption struct {
	ID string `help:"Access group rule Id"`
}

func (opts *AccessGroupRuleIdOption) GetId() string {
	return opts.ID
}

func (opts *AccessGroupRuleIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type AccessRuleCreateOptions struct {
	Priority       int
	Source         string
	RWAccessType   string `choices:"rw|r"`
	UserAccessType string `choices:""`
	Description    string
	AccessGroupId  string
}

func (opts *AccessRuleCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
