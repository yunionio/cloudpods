package types

const (
	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
)

var (
	NIC_TYPES = []string{NIC_TYPE_IPMI, NIC_TYPE_ADMIN}
)

type NicInfo struct {
	Nics []Nic `json:"nic_info"`
}

type Nic struct {
	Type    string `json:"nic_type"`
	Domain  string `json:"domain"`
	Wire    string `json:"wire"`
	IpAddr  string `json:"ip_addr"`
	WireId  string `json:"wire_id"`
	NetId   string `json:"net_id"`
	Rate    int64  `json:"rate"`
	Mtu     int64  `json:"mtu"`
	Mac     string `json:"mac"`
	Dns     string `json:"dns"`
	MaskLen int8   `json:"masklen"`
	Net     string `json:"net"`
	Gateway string `json:"gateway"`
	LinkUp  bool   `json:"link_up"`
}
