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

package proxy

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	ProxySettingId_DIRECT = "DIRECT"
)

type ProxySettingCreateInput struct {
	apis.InfrasResourceBaseCreateInput

	ProxySetting
}

type ProxySettingUpdateInput struct {
	apis.InfrasResourceBaseUpdateInput

	ProxySetting
}

// String implements ISerializable interface
func (ps *SProxySetting) String() string {
	return jsonutils.Marshal(ps).String()
}

// IsZero implements ISerializable interface
func (ps *SProxySetting) IsZero() bool {
	if ps.HTTPProxy == "" &&
		ps.HTTPSProxy == "" &&
		ps.NoProxy == "" {
		return true
	}
	return false
}

type ProxySettingResourceInput struct {
	// 代理配置
	ProxySettingId string `json:"proxy_setting_id"`

	// swagger:ignore
	// Deprecated
	ProxySetting string `json:"proxy_setting" yunion-deprecated-by:"proxy_setting_id"`
}

type ProxySettingTestInput struct {
	HttpProxy  string
	HttpsProxy string
}
