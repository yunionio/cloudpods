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
	"strings"

	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

type LoadbalancerCreateOptions struct {
	NAME             string
	Vpc              string
	Network          string
	Address          string
	AddressType      string `choices:"intranet|internet"`
	LoadbalancerSpec string `choices:"slb.s1.small|slb.s2.small|slb.s2.medium|slb.s3.small|slb.s3.medium|slb.s3.large"`
	ChargeType       string `choices:"traffic|bandwidth"`
	Bandwidth        int
	Zone             string
	Zone1            string `json:"zone_1" help:"slave zone 1"`
	Cluster          string `json:"cluster_id"`
	Manager          string
	Tags             []string `help:"Tags info,prefix with 'user:', eg: user:project=default" json:"-"`
}

func (opts *LoadbalancerCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
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

type LoadbalancerGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	Cluster      string `json:"cluster_id"`
	BackendGroup string
}

type LoadbalancerDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerPurgeOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListOptions struct {
	BaseListOptions
	Address      string
	AddressType  string `choices:"intranet|internet"`
	NetworkType  string `choices:"classic|vpc"`
	Network      string
	BackendGroup string
	Cloudregion  string
	Zone         string
	Cluster      string `json:"cluster_id"`
}

type LoadbalancerActionStatusOptions struct {
	ID     string `json:"-"`
	Status string `choices:"enabled|disabled"`
}

type LoadbalancerActionSyncStatusOptions struct {
	ID string `json:"-"`
}

type LoadbalancerIdOptions struct {
	ID string `json:"-"`
}

type LoadbalancerRemoteUpdateOptions struct {
	ID string `json:"-"`
	computeapi.LoadbalancerRemoteUpdateInput
}
