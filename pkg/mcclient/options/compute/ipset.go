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

type IpSetListOptions struct {
	options.BaseListOptions

	IPSetType []string `help:"filter by ip set type" choices:"ipv4_cidr_list|ipv6_cidr_list" json:"ip_set_type"`
	Ip        string   `help:"filter by ip or cidr" json:"ip"`
}

func (opts *IpSetListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type IpSetCreateOptions struct {
	options.BaseCreateOptions

	IPSetType string `help:"ip set type" choices:"ipv4_cidr_list|ipv6_cidr_list" json:"ip_set_type" required:"true"`
	Data      string `help:"comma separated ip/cidr list, e.g. 192.168.1.0/24,10.0.0.1" json:"data" required:"true"`
}

func (opts *IpSetCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type IpSetIdOptions struct {
	options.BaseIdOptions
}

type IpSetUpdateOptions struct {
	options.BaseIdOptions

	Name      string `help:"new name of ipset"`
	Desc      string `help:"description" json:"description"`
	IPSetType string `help:"ip set type" choices:"ipv4_cidr_list|ipv6_cidr_list" json:"ip_set_type"`
	Data      string `help:"comma separated ip/cidr list, e.g. 192.168.1.0/24,10.0.0.1" json:"data"`
}

func (opts *IpSetUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Name) > 0 {
		params.Add(jsonutils.NewString(opts.Name), "name")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	if len(opts.IPSetType) > 0 {
		params.Add(jsonutils.NewString(opts.IPSetType), "ip_set_type")
	}
	if len(opts.Data) > 0 {
		params.Add(jsonutils.NewString(opts.Data), "data")
	}
	return params, nil
}
