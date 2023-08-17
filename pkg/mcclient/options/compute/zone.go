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

type ZoneListOptions struct {
	options.BaseListOptions
	Region    string `help:"cloud region ID or Name"`
	City      string `help:"Filter zone by city of cloudregions"`
	Usable    *bool  `help:"List all zones where networks are usable"`
	UsableVpc *bool  `help:"List all zones where vpc are usable"`

	OrderByWires             string
	OrderByHosts             string
	OrderByHostsEnabled      string
	OrderByBaremetals        string
	OrderByBaremetalsEnabled string
}

func (opts *ZoneListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ZoneIdOptions struct {
	ID string `help:"ID or Name of zone to update"`
}

func (opts *ZoneIdOptions) GetId() string {
	return opts.ID
}

func (opts *ZoneIdOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

type ZoneUpdateOptions struct {
	ZoneIdOptions
	Name     string `help:"Name of zone"`
	NameCN   string `help:"Name in Chinese"`
	Desc     string `metavar:"<DESCRIPTION>" help:"Description"`
	Location string `help:"Location"`
}

func (opts *ZoneUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Name) > 0 {
		params.Add(jsonutils.NewString(opts.Name), "name")
	}
	if len(opts.NameCN) > 0 {
		params.Add(jsonutils.NewString(opts.NameCN), "name_cn")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	if len(opts.Location) > 0 {
		params.Add(jsonutils.NewString(opts.Location), "location")
	}
	return params, nil
}

type ZoneCapabilityOptions struct {
	ZoneIdOptions
	Domain string
}

func (opts *ZoneCapabilityOptions) Params() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	if len(opts.Domain) > 0 {
		ret.Add(jsonutils.NewString(opts.Domain), "domain")
	}
	return ret, nil
}

type ZoneCreateOptions struct {
	NAME     string `help:"Name of zone"`
	NameCN   string `help:"Name in Chinese"`
	Desc     string `metavar:"<DESCRIPTION>" help:"Description"`
	Location string `help:"Location"`
	Region   string `help:"Cloudregion in which zone created"`
}

func (opts *ZoneCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(opts.NAME), "name")
	if len(opts.NameCN) > 0 {
		params.Add(jsonutils.NewString(opts.NameCN), "name_cn")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	if len(opts.Location) > 0 {
		params.Add(jsonutils.NewString(opts.Location), "location")
	}
	if len(opts.Region) > 0 {
		params.Add(jsonutils.NewString(opts.Region), "region")
	}
	return params, nil
}

type ZoneStatusOptions struct {
	ZoneIdOptions
	STATUS string `help:"zone status" choices:"enable|disable|soldout"`
	REASON string `help:"why update status"`
}

func (opts *ZoneStatusOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("status", jsonutils.NewString(opts.STATUS))
	params.Set("reason", jsonutils.NewString(opts.REASON))
	return params, nil
}

type ZonePurgeOptions struct {
	ZoneIdOptions
	MANAGER_ID string
}

func (opts *ZonePurgeOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("manager_id", jsonutils.NewString(opts.MANAGER_ID))
	return params, nil
}
