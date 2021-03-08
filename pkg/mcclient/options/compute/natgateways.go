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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type NatGatewayListOptions struct {
	options.BaseListOptions
	Vpc         string `help:"vpc id or name"`
	Cloudregion string `help:"cloudreigon id or name"`
}

func (opts *NatGatewayListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type NatGatewayIdOptions struct {
	ID string `help:"ID of Nat Gateway"`
}

func (opts *NatGatewayIdOptions) GetId() string {
	return opts.ID
}

func (opts *NatGatewayIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type NatGatewayDeleteOption struct {
	NatGatewayIdOptions
	Force bool
}

func (opts *NatGatewayDeleteOption) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]bool{"force": opts.Force}), nil
}

type NatDTableListOptions struct {
	Natgateway string `help:"Natgateway name or id"`

	options.BaseListOptions
}

type NatSTableListOptions struct {
	Natgateway string `help:"Natgateway name or id"`
	Network    string `help:"Network id or name"`

	options.BaseListOptions
}

type NatDDeleteShowOptions struct {
	ID string `help:"ID of the DNat"`
}

type NatSDeleteShowOptions struct {
	ID string `help:"ID of the SNat"`
}

type NatDCreateOptions struct {
	NAME         string `help:"DNAT's name"`
	NATGATEWAYID string `help:"The nat gateway'id to which DNat belongs"`
	INTERNALIP   string `help:"Internal IP"`
	INTERNALPORT string `help:"Internal Port"`
	EXTERNALIP   string `help:"External IP"`
	EXTERNALIPID string `help:"External IP ID, can be empty except huawei Cloud"`
	EXTERNALPORT string `help:"External Port"`
	IPPROTOCOL   string `help:"Transport Protocol(tcp|udp)"`
}

type NatSCreateOptions struct {
	NAME         string `help:"SNAT's name"`
	NATGATEWAYID string `help:"The nat gateway'id to which SNat belongs"`
	IP           string `help:"External IP"`
	EXTERNALIPID string `help:"External IP ID, can be empty except huawei Cloud"`
	SourceCIDR   string `help:"Source CIDR"`
	NetworkID    string `help:"Network id"`
}

type NatPostpaidExpireOptions struct {
	NatGatewayIdOptions
	apis.PostpaidExpireInput
}

func (opts *NatPostpaidExpireOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.PostpaidExpireInput), nil
}
