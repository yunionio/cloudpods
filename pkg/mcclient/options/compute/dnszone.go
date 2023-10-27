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

type SDnsZoneListOptions struct {
	options.BaseListOptions

	VpcId     string `help:"Filter dns zone by vpc"`
	ZoneType  string `help:"Filter dns zone by zone type" choices:"PublicZone|PrivateZone"`
	WithCache bool   `help:"Whether to bring cache information"`
}

func (opts *SDnsZoneListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type SDnsZoneIdOptions struct {
	ID string `help:"Dns zone Id or Name"`
}

func (opts *SDnsZoneIdOptions) GetId() string {
	return opts.ID
}

func (opts *SDnsZoneIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DnsZoneCreateOptions struct {
	options.EnabledStatusCreateOptions
	ZoneType  string   `choices:"PublicZone|PrivateZone" metavar:"zone_type" default:"PrivateZone"`
	VpcIds    []string `help:"Vpc Ids"`
	ManagerId string   `help:"Manager id"`
}

func (opts *DnsZoneCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type DnsZoneCapabilitiesOptions struct {
}

func (opts *DnsZoneCapabilitiesOptions) GetId() string {
	return "capability"
}

func (opts *DnsZoneCapabilitiesOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DnsZoneAddVpcsOptions struct {
	SDnsZoneIdOptions
	VPC_IDS string
}

func (opts *DnsZoneAddVpcsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"vpc_ids": opts.VPC_IDS}), nil
}

type DnsZoneRemoveVpcsOptions struct {
	SDnsZoneIdOptions
	VPC_IDS string
}

func (opts *DnsZoneRemoveVpcsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"vpc_ids": opts.VPC_IDS}), nil
}
