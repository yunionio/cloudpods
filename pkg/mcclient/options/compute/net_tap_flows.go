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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type NetTapFlowCreateOptions struct {
	api.NetTapFlowCreateInput
}

func (o *NetTapFlowCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type NetTapFlowListOptions struct {
	options.BaseListOptions

	TapId string `help:"filter by tap id" json:"tap_id"`
}

func (o *NetTapFlowListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type NetTapFlowIdOptions struct {
	ID string `json:"-" help:"Id or name of net tap service"`
}

func (o *NetTapFlowIdOptions) GetId() string {
	return o.ID
}

func (o *NetTapFlowIdOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
