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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ForwardCreateOptions struct {
	NAME string

	ProxyEndpointId string `required:"true"`
	ProxyAgentId    string
	BindPortReq     *int

	Type       string `choices:"local|remote" required:"true"`
	RemoteAddr string `required:"true"`
	RemotePort int    `required:"true"`

	LastSeenTimeout int `json:",omitzero"`

	Opaque string
}

func (o *ForwardCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ForwardCreateFromServerOptions struct {
	ServerId string `required:"true"`

	Type        string `choices:"local|remote" required:"true"`
	BindPortReq int    `json:",omitzero"`
	RemotePort  string `required:"true"`

	LastSeenTimeout int `json:",omitzero"`

	Opaque string
}

func (opts *ForwardCreateFromServerOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type ForwardHeartbeatOptions struct {
	ID string `json:"-"`
}

func (o *ForwardHeartbeatOptions) GetId() string {
	return o.ID
}

func (o *ForwardHeartbeatOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ForwardShowOptions struct {
	options.BaseShowOptions
}

type ForwardUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	ProxyEndpointId string
	ProxyAgentId    string

	Type          string `choices:"local|remote"`
	RemoteAddr    string
	RemotePortReq *int
	BindPortReq   *int

	LastSeenTimeout int `json:",omitzero"`

	Opaque string
}

func (o *ForwardUpdateOptions) GetId() string {
	return o.ID
}

func (o *ForwardUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ForwardDeleteOptions struct {
	options.BaseIdOptions
}

type ForwardListOptions struct {
	options.BaseListOptions

	ProxyEndpointId string
	ProxyAgentId    string

	Type        string `choices:"local|remote"`
	RemoteAddr  string
	RemotePort  *int
	BindPortReq *int

	Opaque string
}

func (o *ForwardListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}
