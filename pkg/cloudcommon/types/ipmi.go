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

package types

import "yunion.io/x/jsonutils"

const (
	POWER_STATUS_ON  = "on"
	POWER_STATUS_OFF = "off"
)

type SIPMIInfo struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	IpAddr     string `json:"ip_addr"`
	Present    bool   `json:"present"`
	LanChannel int    `json:"lan_channel"`
}

func (info SIPMIInfo) ToPrepareParams() jsonutils.JSONObject {
	data := jsonutils.NewDict()
	if info.Username != "" {
		data.Add(jsonutils.NewString(info.Username), "ipmi_username")
	}
	if info.Password != "" {
		data.Add(jsonutils.NewString(info.Password), "ipmi_password")
	}
	if info.IpAddr != "" {
		data.Add(jsonutils.NewString(info.IpAddr), "ipmi_ip_addr")
	}
	data.Add(jsonutils.NewBool(info.Present), "ipmi_present")
	data.Add(jsonutils.NewInt(int64(info.LanChannel)), "ipmi_lan_channel")
	return data
}
