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

import "yunion.io/x/jsonutils"

type VpcPeeringConnectionListOptions struct {
	BaseListOptions
}

func (opts *VpcPeeringConnectionListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type VpcPeeringConnectionIdOptions struct {
	ID string `json:"Vpc peering connection ID"`
}

func (opts *VpcPeeringConnectionIdOptions) GetId() string {
	return opts.ID
}

func (opts *VpcPeeringConnectionIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type VpcPeeringConnectionCreateOptions struct {
	EnabledStatusCreateOptions
	VpcId     string
	PeerVpcId string
	Bandwidth int
}

func (opts *VpcPeeringConnectionCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts).(*jsonutils.JSONDict), nil
}
