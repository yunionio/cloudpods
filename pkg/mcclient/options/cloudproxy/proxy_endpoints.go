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

package cloudproxy

import (
	"io/ioutil"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ProxyEndpointCreateOptions struct {
	NAME string

	User       string
	Host       string
	Port       int `json:",omitzero"`
	PrivateKey string

	IntranetIpAddr string `required:"true"`
}

func (opts *ProxyEndpointCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	if opts.PrivateKey != "" && opts.PrivateKey[0] == '@' {
		data, err := ioutil.ReadFile(opts.PrivateKey[1:])
		if err != nil {
			return nil, err
		}
		params.Set("private_key", jsonutils.NewString(string(data)))
	}
	return params, nil
}

type ProxyEndpointCreateFromServerOptions struct {
	Name string

	ServerId string `required:"true"`
}

func (opts *ProxyEndpointCreateFromServerOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type ProxyEndpointShowOptions struct {
	options.BaseShowOptions
}

type ProxyEndpointUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	User       string
	Host       string
	Port       int `json:",omitzero"`
	PrivateKey string
}

func (opts *ProxyEndpointUpdateOptions) GetId() string {
	return opts.ID
}

func (opts *ProxyEndpointUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	if opts.PrivateKey != "" && opts.PrivateKey[0] == '@' {
		data, err := ioutil.ReadFile(opts.PrivateKey[1:])
		if err != nil {
			return nil, err
		}
		params.Set("private_key", jsonutils.NewString(string(data)))
	}
	return params, nil
}

type ProxyEndpointDeleteOptions struct {
	options.BaseShowOptions
}

type ProxyEndpointListOptions struct {
	options.BaseListOptions

	//User       string
	Host string
	Port int `json:",omitzero"`

	VpcId     string
	NetworkId string
}

func (opts *ProxyEndpointListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}
