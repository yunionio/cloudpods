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
	"fmt"

	"yunion.io/x/jsonutils"
)

type SkuSyncOptions struct {
	// 云平台名称
	// example: Google
	Provider string `json:"provider,omitempty" help:"cloud provider name"`

	// 区域ID
	CloudregionIds []string `json:"cloudregion_ids" help:"cloud region id list"`
}

func (opts *SkuSyncOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SkuTaskQueryOptions struct {
	// 异步任务ID
	TaskIds []string `json:"task_ids" help:"task ids"`
}

func (opts *SkuTaskQueryOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.TaskIds) == 0 {
		return nil, fmt.Errorf("task_ids is empty")
	}

	params := jsonutils.NewDict()
	params.Set("task_ids", jsonutils.Marshal(opts.TaskIds))
	return params, nil
}

type CloudregionSkuSyncOptions struct {
	RESOURCE string `help:"Resource of skus" choices:"serversku|elasticcachesku|dbinstance_sku"`
	SkuSyncOptions
}

func (opts *CloudregionSkuSyncOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type CloudregionIdOptions struct {
	ID string `help:"Cloudregion Id"`
}

func (opts *CloudregionIdOptions) GetId() string {
	return opts.ID
}

func (opts *CloudregionIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
