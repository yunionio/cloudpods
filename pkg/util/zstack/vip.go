package zstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SVirtualIP struct {
	ZStackBasic
	IPRangeUUID        string   `json:"ipRangeUuid"`
	L3NetworkUUID      string   `json:"l3NetworkUuid"`
	IP                 string   `json:"ip"`
	State              string   `json:"state"`
	Gateway            string   `json:"gateway"`
	Netmask            string   `json:"netmask"`
	PrefixLen          int      `json:"prefixLen"`
	ServiceProvider    string   `json:"serviceProvider"`
	PeerL3NetworkUuids []string `json:"peerL3NetworkUuids"`
	UseFor             string   `json:"useFor"`
	UsedIPUUID         string   `json:"usedIpUuid"`
	ZStackTime
}

func (region *SRegion) GetVirtualIP(vipId string) (*SVirtualIP, error) {
	vip := &SVirtualIP{}
	return vip, region.client.getResource("vips", vipId, vip)
}

func (region *SRegion) GetNetworkId(vip *SVirtualIP) string {
	networks, err := region.GetNetworks("", "", vip.L3NetworkUUID, "")
	if err == nil {
		for _, network := range networks {
			if network.Contains(vip.IP) {
				return fmt.Sprintf("%s/%s", vip.L3NetworkUUID, network.UUID)
			}
		}
	}
	return ""
}

func (region *SRegion) GetVirtualIPs(vipId string) ([]SVirtualIP, error) {
	vips := []SVirtualIP{}
	params := []string{}
	if len(vipId) > 0 {
		params = append(params, "q=uuid="+vipId)
	}
	return vips, region.client.listAll("vips", params, &vips)
}

func (region *SRegion) CreateVirtualIP(name, desc, ip string, l3Id string) (*SVirtualIP, error) {
	vip := SVirtualIP{}
	params := map[string]map[string]string{
		"params": {
			"name":          name,
			"description":   desc,
			"l3NetworkUuid": l3Id,
		},
	}
	if len(ip) > 0 {
		params["params"]["requiredIp"] = ip
	}
	resp, err := region.client.post("vips", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	return &vip, resp.Unmarshal(&vip, "inventory")
}

func (region *SRegion) DeleteVirtualIP(vipId string) error {
	return region.client.delete("vips", vipId, "")
}
