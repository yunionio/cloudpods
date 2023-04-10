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

type WireListOptions struct {
	BaseListOptions

	Bandwidth *int `help:"List wires by bandwidth"`

	Region   string `help:"List wires in region"`
	Zone     string `help:"list wires in zone" json:"-"`
	Vpc      string `help:"List wires in vpc"`
	Host     string `help:"List wires attached to a host"`
	HostType string `help:"List wires attached to host with HostType"`

	OrderByNetworkCount string
}

func (wo *WireListOptions) GetContextId() string {
	return wo.Vpc
}

func (wo *WireListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(wo)
}

type WireOptions struct {
	ID string `help:"Id or Name of wire to update"`
}

func (wo *WireOptions) GetId() string {
	return wo.ID
}

func (wo *WireOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type WireUpdateOptions struct {
	WireOptions
	SwireupdateOptions
}

type SwireupdateOptions struct {
	Name string `help:"Name of wire" json:"name"`
	Desc string `metavar:"<DESCRIPTION>" help:"Description" json:"description"`
	Bw   int64  `help:"Bandwidth in mbps" json:"bandwidth"`
	Mtu  int64  `help:"mtu in bytes" json:"mtu"`
}

func (wo *WireUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(wo.SwireupdateOptions), nil
}

type WireCreateOptions struct {
	ZONE string `help:"Zone ID or Name"`
	Vpc  string `help:"VPC ID or Name" default:"default"`
	NAME string `help:"Name of wire"`
	BW   int64  `help:"Bandwidth in mbps"`
	Mtu  int64  `help:"mtu in bytes"`
	Desc string `metavar:"<DESCRIPTION>" help:"Description"`
}

func (wo *WireCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(wo), nil
}

type SwirePublicOptions struct {
	Scope         string   `help:"sharing scope" choices:"system|domain"`
	SharedDomains []string `help:"share to domains"`
}

type WirePublicOptions struct {
	WireOptions
	SwirePublicOptions
}

func (wo *WirePublicOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(wo.SwirePublicOptions), nil
}

type WireMergeOptions struct {
	Froms        []string `help:"IDs or names of merge wire from" json:"sources"`
	TARGET       string   `help:"ID or name of merge wire target"`
	MergeNetwork bool     `help:"whether to merge network under wire"`
}

func (wo *WireMergeOptions) GetId() string {
	return wo.TARGET
}

func (wo *WireMergeOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(wo), nil
}
