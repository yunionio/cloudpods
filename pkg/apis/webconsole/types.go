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

package webconsole

import (
	"encoding/base64"
	"net/url"

	"yunion.io/x/onecloud/pkg/apis"
)

type ServerRemoteConsoleResponse struct {
	AccessUrl     string `json:"access_url"`
	ConnectParams string `json:"connect_params"`
	Session       string `json:"session,omitempty"`

	apis.Meta
}

func (resp *ServerRemoteConsoleResponse) GetConnectParams() string {
	params := resp.ConnectParams
	if data, err := base64.StdEncoding.DecodeString(params); err == nil {
		params = string(data)
	}
	return params
}

func (resp *ServerRemoteConsoleResponse) GetConnectProtocol() (string, error) {
	var (
		params = resp.GetConnectParams()
		query  url.Values
	)
	if uri, err := url.ParseRequestURI(params); err == nil {
		query = uri.Query()
	} else if q, err := url.ParseQuery(params); err == nil {
		query = q
	} else {
		return "", err
	}
	protocol := query.Get("protocol")
	return protocol, nil
}
