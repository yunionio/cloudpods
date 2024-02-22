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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LoadbalancerCreateOptions struct {
	NAME             string
	Vpc              string
	Network          string
	Address          string
	AddressType      string `choices:"intranet|internet"`
	LoadbalancerSpec string `choices:"slb.s1.small|slb.s2.small|slb.s2.medium|slb.s3.small|slb.s3.medium|slb.s3.large|network"`
	ChargeType       string `choices:"traffic|bandwidth"`
	Bandwidth        int
	Zone             string
	Zone1            string `json:"zone_1" help:"slave zone 1"`
	Cluster          string `json:"cluster_id"`
	Manager          string
	Tags             []string `help:"Tags info,prefix with 'user:', eg: user:project=default" json:"-"`

	Eip              string `json:"eip" help:"Id or name of EIP to associate with"`
	EipBw            int    `json:"eip_bw"`
	EipChargeType    string `json:"eip_charge_type"`
	EipBgpType       string `json:"eip_bgp_type"`
	EipAutoDellocate *bool  `json:"eip_auto_dellocate"`
}

func (opts *LoadbalancerCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	Tagparams := jsonutils.NewDict()
	for _, tag := range opts.Tags {
		info := strings.Split(tag, "=")
		if len(info) == 2 {
			if len(info[0]) == 0 {
				return nil, fmt.Errorf("invalidate tag info %s", tag)
			}
			Tagparams.Add(jsonutils.NewString(info[1]), info[0])
		} else if len(info) == 1 {
			Tagparams.Add(jsonutils.NewString(info[0]), info[0])
		} else {
			return nil, fmt.Errorf("invalidate tag info %s", tag)
		}
	}
	params.Add(Tagparams, "__meta__")
	return params, nil
}

type LoadbalancerIdOptions struct {
	ID string `json:"-"`
}

func (opts *LoadbalancerIdOptions) GetId() string {
	return opts.ID
}

func (opts *LoadbalancerIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type LoadbalancerUpdateOptions struct {
	LoadbalancerIdOptions
	Name string

	Delete       string `help:"Lock server to prevent from deleting" choices:"enable|disable" json:"-"`
	Cluster      string `json:"cluster_id"`
	BackendGroup string
}

func (opts LoadbalancerUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	if len(opts.Delete) > 0 {
		if opts.Delete == "disable" {
			params.Set("disable_delete", jsonutils.JSONTrue)
		} else {
			params.Set("disable_delete", jsonutils.JSONFalse)
		}
	}
	return params, nil
}

type LoadbalancerDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerPurgeOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListOptions struct {
	options.BaseListOptions

	Address      string
	AddressType  string `choices:"intranet|internet"`
	NetworkType  string `choices:"classic|vpc"`
	Network      string
	BackendGroup string
	Cloudregion  string
	Zone         string
	Cluster      string `json:"cluster_id"`
	SecgroupId   string
}

func (opts *LoadbalancerListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type LoadbalancerActionStatusOptions struct {
	LoadbalancerIdOptions
	Status string `choices:"enabled|disabled"`
}

func (opts *LoadbalancerActionStatusOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.Status) == 0 {
		return nil, fmt.Errorf("empty status")
	}
	return jsonutils.Marshal(map[string]string{"status": opts.Status}), nil
}

type LoadbalancerActionSyncStatusOptions struct {
	ID string `json:"-"`
}

type LoadbalancerRemoteUpdateOptions struct {
	LoadbalancerIdOptions
	computeapi.LoadbalancerRemoteUpdateInput
}

func (opts *LoadbalancerRemoteUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type LoadbalancerAssociateEipOptions struct {
	LoadbalancerIdOptions
	computeapi.LoadbalancerAssociateEipInput
}

func (opts *LoadbalancerAssociateEipOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type LoadbalancerCreateEipOptions struct {
	LoadbalancerIdOptions
	computeapi.LoadbalancerCreateEipInput
}

func (opts *LoadbalancerCreateEipOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type LoadbalancerDissociateEipOptions struct {
	LoadbalancerIdOptions
	computeapi.LoadbalancerDissociateEipInput
}

func (opts *LoadbalancerDissociateEipOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
