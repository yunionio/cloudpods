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

package cloudnet

import (
	"io/ioutil"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type RouterCreateOptions struct {
	NAME       string
	User       string
	Host       string
	Port       int
	PrivateKey string

	RealizeWgIfaces string `choices:"on|off" default:"on" help:"apply wg ifaces config on realization"`
	RealizeRoutes   string `choices:"on|off" default:"on" help:"apply routes config on realization"`
	RealizeRules    string `choices:"on|off" default:"on" help:"apply firewall rules on realization"`
}

func (opts *RouterCreateOptions) Params() (jsonutils.JSONObject, error) {
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

type RouterGetOptions struct {
	ID string `json:"-"`
}

type RouterUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	User       string
	Host       string
	Port       int `json:",omitzero"`
	PrivateKey string

	RealizeWgIfaces string `json:",omitzero" choices:"on|off" help:"apply wg ifaces config on realization"`
	RealizeRoutes   string `json:",omitzero" choices:"on|off" help:"apply routes config on realization"`
	RealizeRules    string `json:",omitzero" choices:"on|off" help:"apply firewall rules on realization"`
}

func (opts *RouterUpdateOptions) Params() (jsonutils.JSONObject, error) {
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

type RouterDeleteOptions struct {
	ID string `json:"-"`
}

type RouterListOptions struct {
	options.BaseListOptions
}

type RouterActionJoinMeshNetworkOptions struct {
	ID string `json:"-"`

	MeshNetwork      string
	AdvertiseSubnets string `help:"cidr concatenated by comma"`
}

type RouterActionLeaveMeshNetworkOptions struct {
	ID string `json:"-"`

	MeshNetwork string
}

type RouterActionRegisterIfnameOptions struct {
	ID string `json:"-"`

	Ifname string
}

type RouterActionUnregisterIfnameOptions struct {
	ID string `json:"-"`

	Ifname string
}

type RouterActionRealizeOptions struct {
	ID string `json:"-"`
}
