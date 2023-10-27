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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type NetworkAddressCreateOptions struct {
	ParentType string `help:"object type" choices:"guestnetwork" default:"guestnetwork"`
	ParentId   string `help:"object id"`

	Guest             string `help:"guest name or uuid" json:"guest_id"`
	GuestnetworkIndex int    `help:"guest network interface index"`

	Type   string `help:"address type" choices:"sub_ip" default:"sub_ip"`
	IPAddr string `help:"preferred ip address"`

	Count int `help:"create multiple addresses" json:",omitzero"`
}

func (opts *NetworkAddressCreateOptions) Params() (jsonutils.JSONObject, error) {
	if opts.Guest != "" {
	} else if opts.ParentId != "" {
		return nil, errors.Error("do not support directly specifying parent id yet")
	} else {
		return nil, errors.Error("requires parent spec, try --guest, etc.")
	}
	return StructToParams(opts)
}

func (opts *NetworkAddressCreateOptions) GetCountParam() int {
	return opts.Count
}

type NetworkAddressListOptions struct {
	BaseListOptions

	Type       string
	ParentType string
	ParentId   string
	NetworkId  string
	IpAddr     string

	GuestId []string
}

func (opts *NetworkAddressListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type NetworkAddressIdOptions struct {
	ID string
}

func (opts *NetworkAddressIdOptions) GetId() string {
	return opts.ID
}

func (opts *NetworkAddressIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type NetworkAddressIdsOptions struct {
	ID []string
}

func (opts *NetworkAddressIdsOptions) GetIds() []string {
	return opts.ID
}

func (opts *NetworkAddressIdsOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
