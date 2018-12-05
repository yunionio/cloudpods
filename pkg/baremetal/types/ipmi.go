package types

type IPMIInfo struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	IpAddr     string `json:"ip_addr"`
	Present    bool   `json:"present"`
	LanChannel int    `json:"lan_channel"`
}
