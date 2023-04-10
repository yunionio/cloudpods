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

type InterVpcNetworkListOPtions struct {
	BaseListOptions

	OrderByVpcCount string
}

func (opts *InterVpcNetworkListOPtions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type InterVpcNetworkIdOPtions struct {
	ID string `json:"Vpc peering connection ID"`
}

func (opts *InterVpcNetworkIdOPtions) GetId() string {
	return opts.ID
}

func (opts *InterVpcNetworkIdOPtions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type InterVpcNetworkCreateOPtions struct {
	EnabledStatusCreateOptions
	ManagerId string
}

func (opts *InterVpcNetworkCreateOPtions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts).(*jsonutils.JSONDict), nil
}

type InterVpcNetworkAddVpcOPtions struct {
	ID    string `json:"Vpc peering connection ID"`
	VpcId string `json:"Vpc ID"`
}

func (opts *InterVpcNetworkAddVpcOPtions) GetId() string {
	return opts.ID
}

func (opts *InterVpcNetworkAddVpcOPtions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts).(*jsonutils.JSONDict), nil
}

type InterVpcNetworkRemoveVpcOPtions struct {
	ID    string `json:"Vpc peering connection ID"`
	VpcId string `json:"Vpc ID"`
}

func (opts *InterVpcNetworkRemoveVpcOPtions) GetId() string {
	return opts.ID
}

func (opts *InterVpcNetworkRemoveVpcOPtions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts).(*jsonutils.JSONDict), nil
}
