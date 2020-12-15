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
)

type ServerOpenForwardOptions struct {
	ServerIdOptions
	Proto string `json:"proto" help:"protocol" choices:"tcp|udp" default:"tcp"`
	Port  int    `json:"port" help:"port" required:"true"`
}

func (o *ServerOpenForwardOptions) Params() (jsonutils.JSONObject, error) {
	return StructToParams(o)
}

type ServerCloseForwardOptions struct {
	ServerIdOptions
	Proto     string `json:"proto" help:"protocol" choices:"tcp|udp" default:"tcp"`
	ProxyAddr string `json:"proxy_addr" help:"proxy addr" required:"true"`
	ProxyPort int    `json:"proxy_port" help:"proxy port" required:"true"`
}

func (o *ServerCloseForwardOptions) Params() (jsonutils.JSONObject, error) {
	return StructToParams(o)
}

type ServerListForwardOptions struct {
	ServerIdOptions
	Proto *string `json:"proto" help:"protocol" choices:"tcp|udp"`
	Port  *int    `json:"port" help:"port"`
}

func (o *ServerListForwardOptions) Params() (jsonutils.JSONObject, error) {
	return StructToParams(o)
}
