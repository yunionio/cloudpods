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

type ElasticCacheCreateOptions struct {
	NAME          string
	Manager       string
	Cloudregion   string
	Zone          string
	VpcId         string
	Network       string   `help:"network id"`
	SecurityGroup string   `help:"elastic cache security group. required by huawei."`
	SecgroupIds   []string `help:"elastic cache security group. required by qcloud."`
	Engine        string   `choices:"redis"`
	EngineVersion string   `choices:"2.8|3.0|3.2|4.0|5.0"`
	PrivateIP     string   `help:"private ip address in specificated network"`
	Password      string   `help:"set auth password"`
	InstanceType  string
	CapacityMB    string   `help:"elastic cache capacity. required by huawei."`
	BillingType   string   `choices:"postpaid|prepaid" default:"postpaid"`
	Month         int      `help:"billing duration (unit:month)"`
	Tags          []string `help:"Tags info,prefix with 'user:', eg: user:project=default" json:"-"`
}

func (opts *ElasticCacheCreateOptions) Params() (*jsonutils.JSONDict, error) {
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

type ElasticCacheAccountCreateOptions struct {
	Elasticcache     string `help:"elastic cache instance id"`
	Name             string
	Password         string
	AccountPrivilege string `help:"account privilege" choices:"read|write|repl" default:"read"`
}

type ElasticCacheBackupCreateOptions struct {
	Elasticcache string `help:"elastic cache instance id"`
	Name         string
}

type ElasticCacheAclCreateOptions struct {
	Elasticcache string `help:"elastic cache instance id"`
	Name         string
	IpList       string `help:"elastic cache acl ip list, split by ','"`
}

type ElasticCacheAclUpdateOptions struct {
	Id     string `help:"elastic cache acl id"`
	IpList string `help:"elastic cache acl ip list, split by ','"`
}

type ElasticCacheParameterUpdateOptions struct {
	Id    string `help:"elastic cache parameter id"`
	Value string `help:"elastic cache parameter value"`
}

type ElasticCacheIdOptions struct {
	ID string
}

type ElasticCacheRemoteUpdateOptions struct {
	ID string `json:"-"`
	computeapi.ElasticcacheRemoteUpdateInput
}

type ElasticCacheAutoRenewOptions struct {
	ElasticCacheIdOptions
	AutoRenew bool `help:"Set elastic cache auto renew or manual renew"`
}

func (o *ElasticCacheAutoRenewOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("auto_renew", jsonutils.NewBool(o.AutoRenew))
	return params, nil
}

type ElasticCacheRenewOptions struct {
	ID       string `json:"-"`
	Duration string `help:"valid duration of the elastic cache, e.g. 1H, 1D, 1W, 1M, 1Y, ADMIN ONLY option"`
}
