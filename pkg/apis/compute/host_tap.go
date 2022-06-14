package compute

const (
	HostVpcBridge = "__vpc_bridge__"
	HostTapBridge = "__tap_bridge__"
)

type SMirrorConfig struct {
	TapHostIp string `json:"tap_host_ip"`

	HostIp string `json:"host_ip"`

	Port string `json:"port"`

	Bridge string `json:"bridge"`

	FlowId uint16 `json:"flow_id"`

	VlanId int `json:"vlan_id"`

	Direction string `json:"direction"`
}

type SHostBridgeMirrorConfig struct {
	TapHostIp string `json:"tap_host_ip"`

	HostIp string `json:"host_ip"`

	Bridge string `json:"bridge"`

	Direction string `json:"direction"`

	FlowId uint16 `json:"flow_id"`

	VlanId []int `json:"vlan_id"`

	Port []string `json:"port"`
}

type STapServiceConfig struct {
	TapHostIp string `json:"tap_host_ip"`

	MacAddr string `json:"mac_addr"`

	Ifname string `json:"ifname"`

	Mirrors []SHostBridgeMirrorConfig
}

type SHostTapConfig struct {
	Taps []STapServiceConfig `json:"taps"`

	Mirrors []SHostBridgeMirrorConfig `json:"mirrors"`
}
