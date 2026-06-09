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

package aiproxy

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type AiProxyNodeListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	Address string `json:"address"`
	Domain  string `json:"domain"`
}

type AiProxyNodeCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	Address   string `json:"address"`
	Domain    string `json:"domain"`
	HbTimeout int    `json:"hb_timeout"`
}

type AiProxyNodeUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	Address   string `json:"address"`
	Domain    string `json:"domain"`
	HbTimeout int    `json:"hb_timeout"`
	Enabled   *bool  `json:"enabled"`
}

type AiProxyNodeDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	Address   string    `json:"address"`
	Domain    string    `json:"domain"`
	LastSeen  time.Time `json:"last_seen"`
	HbTimeout int       `json:"hb_timeout"`
	IsActive  bool      `json:"is_active"`
}

// AiProxyNodeRegisterInput is sent by standby instances to the primary on startup and heartbeat.
type AiProxyNodeRegisterInput struct {
	Address   string `json:"address"`
	HbTimeout int    `json:"hb_timeout"`
}

type AiProxyNodeRegisterOutput struct {
	Id string `json:"id"`
}
