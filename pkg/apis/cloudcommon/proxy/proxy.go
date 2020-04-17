package proxy

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	ProxySettingId_DIRECT = "DIRECT"
)

type ProxySettingCreateInput struct {
	apis.StandaloneResourceCreateInput

	ProxySetting
}

type ProxySettingUpdateInput struct {
	// 资源名称
	Name string `json:"name"`
	// 资源描述
	Description string `json:"description"`

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
	ProxySetting string `json:"proxy_setting"`

	// swagger:ignore
	// Deprecated
	ProxySettingId string `json:"proxy_setting_id" "yunion:deprecated-by":"proxy_setting"`
}

type ProxySettingTestInput struct {
	HttpProxy  string
	HttpsProxy string
}
