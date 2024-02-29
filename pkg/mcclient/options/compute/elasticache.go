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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ElasticCacheListOptions struct {
	options.BaseListOptions

	SecgroupId string
}

func (opts *ElasticCacheListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ElasticCacheIdOption struct {
	ID string `help:"Elasticache Id"`
}

func (opts *ElasticCacheIdOption) GetId() string {
	return opts.ID
}

func (opts *ElasticCacheIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

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

func (opts *ElasticCacheCreateOptions) Params() (jsonutils.JSONObject, error) {
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
	if len(opts.SecgroupIds) > 0 {
		params.Set("secgroup_ids", jsonutils.NewStringArray(opts.SecgroupIds))
	}
	params.Add(Tagparams, "__meta__")
	return params, nil
}

type ElasticCacheChangeSpecOptions struct {
	ElasticCacheIdOption
	Sku string `help:"elastic cache sku id"`
}

func (opts *ElasticCacheChangeSpecOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"sku": opts.Sku}), nil
}

type ElasticCacheMainteananceTimeOptions struct {
	ElasticCacheIdOption
	START_TIME string `help:"elastic cache sku maintenance start time,format: HH:mm"`
	END_TIME   string `help:"elastic cache sku maintenance end time, format: HH:mm"`
}

func (opts *ElasticCacheMainteananceTimeOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	start := fmt.Sprintf("%sZ", opts.START_TIME)
	end := fmt.Sprintf("%sZ", opts.END_TIME)
	params.Set("maintain_start_time", jsonutils.NewString(start))
	params.Set("maintain_end_time", jsonutils.NewString(end))
	return params, nil
}

type ElasticCacheEnableAuthOptions struct {
	ElasticCacheIdOption
}

func (opts *ElasticCacheEnableAuthOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"auth_mod": "on"}), nil
}

type ElasticCacheDisableAuthOptions struct {
	ElasticCacheIdOption
}

func (opts *ElasticCacheDisableAuthOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"auth_mod": "off"}), nil
}
