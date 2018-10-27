package types

type NetworkConfig struct {
	GuestDhcp    string `json:"guest_dhcp"`
	GuestGateway string `json:"guest_gateway"`
	GuestIpStart string `json:"guest_ip_start"`
	GuestIpEnd   string `json:"guest_ip_end"`
	GuestIpMask  int    `json:"guest_ip_mask"`
	Id           string `json:"id"`
	IsEmulated   bool   `json:"is_emulated"`
	IsPublic     bool   `json:"is_public"`
	IsSystem     bool   `json:"is_system"`
	Name         string `json:"name"`
	ServerType   string `json:"server_type"`
	Status       string `json:"status"`
	ProjectId    string `json:"tenant_id"`
	VlanId       int    `json:"vlan_id"`
	WireId       string `json:"wire_id"`
}
