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

package devtool

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ServiceUrlCreateOptions struct {
	Service           string `help:"service name"`
	ServerId          string `help:"server id"`
	ServerAnsibleInfo SServerAnisbleInfo
}

type SServerAnisbleInfo struct {
	User string `json:"user"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Name string `json:"name"`
}

func (so *ServiceUrlCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(so), nil
}

type ServiceUrlListOptions struct {
	options.BaseListOptions
}

func (so *ServiceUrlListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(so)
}

type ServiceUrlOptions struct {
	ID string `help:"id or name of sshinfo"`
}

func (so *ServiceUrlOptions) GetId() string {
	return so.ID
}

func (so *ServiceUrlOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
