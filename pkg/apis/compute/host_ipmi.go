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
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

const (
	// DefaultIpmiPort 是 IPMI RMCP 端口。
	DefaultIpmiPort = 623
	// DefaultRedfishPort 是 Redfish HTTPS 端口。
	DefaultRedfishPort = 443
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SIpmiInfo{}), func() gotypes.ISerializable {
		return &SIpmiInfo{}
	})
}

// SIpmiInfo 是宿主机 IPMI/Redfish 访问信息，按 JSON 存在 hosts 表的 ipmi_info 列。
type SIpmiInfo struct {
	// 用户名
	Username string `json:"username,omitempty"`
	// 密码，库内为 AES 密文
	Password string `json:"password,omitempty"`
	// IP 地址
	IpAddr string `json:"ip_addr,omitempty"`
	// IPMI RMCP 端口，未设置时为 623
	IpmiPort int `json:"ipmi_port,omitzero"`
	// Redfish HTTPS 端口，未设置时为 443
	RedfishPort int `json:"redfish_port,omitzero"`
	// 是否存在 BMC
	Present bool `json:"present,omitempty"`
	// IPMI LAN channel
	LanChannel uint8 `json:"lan_channel,omitzero"`
	// RMCP+ cipher suite
	CipherSuite int `json:"cipher_suite,omitzero"`
	// 是否已探测通过
	Verified bool `json:"verified,omitfalse"`
	// 是否支持 Redfish API
	RedfishApi bool `json:"redfish_api,omitfalse"`
	// 是否支持光驱启动
	CdromBoot bool `json:"cdrom_boot,omitfalse"`
	// 是否支持 PXE 启动
	PxeBoot bool `json:"pxe_boot,omitfalse"`
}

func (info *SIpmiInfo) String() string {
	return jsonutils.Marshal(info).String()
}

func (info *SIpmiInfo) IsZero() bool {
	if info == nil {
		return true
	}
	return *info == SIpmiInfo{}
}

// GetIpmiPort 返回 IPMI 端口。字段未设置或不是合法端口时返回 623。
func (info *SIpmiInfo) GetIpmiPort() int {
	if info == nil || info.IpmiPort <= 0 || info.IpmiPort > 65535 {
		return DefaultIpmiPort
	}
	return info.IpmiPort
}

// GetRedfishPort 返回 Redfish 端口。字段未设置或不是合法端口时返回 443。
func (info *SIpmiInfo) GetRedfishPort() int {
	if info == nil || info.RedfishPort <= 0 || info.RedfishPort > 65535 {
		return DefaultRedfishPort
	}
	return info.RedfishPort
}

// RedfishEndpoint 返回带端口的 Redfish HTTPS 地址。
func (info *SIpmiInfo) RedfishEndpoint() string {
	ip := ""
	port := 0
	if info != nil {
		ip = info.IpAddr
		port = info.RedfishPort
	}
	return RedfishEndpoint(ip, port)
}

// RedfishEndpoint 用 IP 和端口拼 Redfish HTTPS 地址。port 未设置时使用 443。
func RedfishEndpoint(ip string, port int) string {
	if port <= 0 || port > 65535 {
		port = DefaultRedfishPort
	}
	host := ip
	if strings.Contains(ip, ":") {
		host = "[" + ip + "]"
	}
	return fmt.Sprintf("https://%s:%d", host, port)
}

func (info SIpmiInfo) ToPrepareParams() jsonutils.JSONObject {
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
	if info.IpmiPort > 0 {
		data.Add(jsonutils.NewInt(int64(info.IpmiPort)), "ipmi_port")
	}
	if info.RedfishPort > 0 {
		data.Add(jsonutils.NewInt(int64(info.RedfishPort)), "ipmi_redfish_port")
	}
	data.Add(jsonutils.NewBool(info.Present), "ipmi_present")
	data.Add(jsonutils.NewInt(int64(info.LanChannel)), "ipmi_lan_channel")
	if info.CipherSuite > 0 {
		data.Add(jsonutils.NewInt(int64(info.CipherSuite)), "ipmi_cipher_suite")
	}
	if info.Verified {
		data.Add(jsonutils.JSONTrue, "ipmi_verified")
	}
	if info.RedfishApi {
		data.Add(jsonutils.JSONTrue, "ipmi_redfish_api")
	}
	if info.CdromBoot {
		data.Add(jsonutils.JSONTrue, "ipmi_cdrom_boot")
	}
	if info.PxeBoot {
		data.Add(jsonutils.JSONTrue, "ipmi_pxe_boot")
	}
	return data
}
