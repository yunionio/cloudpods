package proxy

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	ProxySettingId_DIRECT = "DIRECT"
)

type ProxySettingCreateInput struct {
	apis.VirtualResourceCreateInput

	HttpProxy  string
	HttpsProxy string
	NoProxy    string
}

type ProxySettingUpdateInput ProxySettingCreateInput

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
