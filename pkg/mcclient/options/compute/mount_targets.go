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

type MountTargetListOptions struct {
	options.BaseListOptions
}

func (opts *MountTargetListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type MountTargetIdOption struct {
	ID string `help:"Mount Target Id"`
}

func (opts *MountTargetIdOption) GetId() string {
	return opts.ID
}

func (opts *MountTargetIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type MountTargetCreateOptions struct {
	NAME          string
	NetworkType   string `choices:"vpc|classic"`
	FileSystemId  string `json:"file_system_id"`
	NetworkId     string `json:"network_id"`
	AccessGroupId string `json:"access_group_id"`
}

func (opts *MountTargetCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
