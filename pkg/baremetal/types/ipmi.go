package types

import "yunion.io/x/jsonutils"

const (
	POWER_STATUS_ON  = "on"
	POWER_STATUS_OFF = "off"
)

type IPMIInfo struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	IpAddr     string `json:"ip_addr"`
	Present    bool   `json:"present"`
	LanChannel int    `json:"lan_channel"`
}

func (info IPMIInfo) ToPrepareParams() jsonutils.JSONObject {
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
