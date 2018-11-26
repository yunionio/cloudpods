package azure

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/netutils"
)

type PublicIPAddress struct {
	ID         string
	Name       string
	Location   string
	Properties PublicIPAddressPropertiesFormat
}

type InterfaceIPConfigurationPropertiesFormat struct {
	PrivateIPAddress          string           `json:"privateIPAddress,omitempty"`
	PrivateIPAddressVersion   string           `json:"privateIPAddressVersion,omitempty"`
	PrivateIPAllocationMethod string           `json:"privateIPAllocationMethod,omitempty"`
	Subnet                    Subnet           `json:"subnet,omitempty"`
	Primary                   *bool            `json:"primary,omitempty"`
	PublicIPAddress           *PublicIPAddress `json:"publicIPAddress,omitempty"`
}

type InterfaceIPConfiguration struct {
	Properties InterfaceIPConfigurationPropertiesFormat `json:"properties,omitempty"`
	Name       string
	ID         string
}

type InterfacePropertiesFormat struct {
	NetworkSecurityGroup *SSecurityGroup            `json:"networkSecurityGroup,omitempty"`
	IPConfigurations     []InterfaceIPConfiguration `json:"ipConfigurations,omitempty"`
	MacAddress           string                     `json:"aacAddress,omitempty"`
	Primary              *bool                      `json:"primary,omitempty"`
	VirtualMachine       *SubResource               `json:"virtualMachine,omitempty"`
}

type SInstanceNic struct {
	instance   *SInstance
	ID         string
	Name       string
	Type       string
	Location   string
	Properties InterfacePropertiesFormat `json:"properties,omitempty"`
}

func (self *SInstanceNic) GetIP() string {
	if len(self.Properties.IPConfigurations) > 0 {
		return self.Properties.IPConfigurations[0].Properties.PrivateIPAddress
	}
	return ""
}

func (region *SRegion) DeleteNetworkInterface(interfaceId string) error {
	return region.client.Delete(interfaceId)
}

func (self *SInstanceNic) Delete() error {
	return self.instance.host.zone.region.DeleteNetworkInterface(self.ID)
}

func (self *SInstanceNic) GetMAC() string {
	mac := self.Properties.MacAddress
	if len(mac) == 0 {
		ip, _ := netutils.NewIPV4Addr(self.GetIP())
		return ip.ToMac("00:16:")
	}
	return strings.Replace(strings.ToLower(mac), "-", ":", -1)
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) updateSecurityGroup(secgroupId string) error {
	region := self.instance.host.zone.region
	self.Properties.NetworkSecurityGroup = nil
	if len(secgroupId) > 0 {
		self.Properties.NetworkSecurityGroup = &SSecurityGroup{ID: secgroupId}
	}
	return region.client.Update(jsonutils.Marshal(self), nil)
}

func (self *SInstanceNic) revokeSecurityGroup() error {
	return self.updateSecurityGroup("")
}

func (self *SInstanceNic) assignSecurityGroup(secgroupId string) error {
	return self.updateSecurityGroup(secgroupId)
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	wires, err := self.instance.host.GetIWires()
	if err != nil {
		log.Errorf("GetINetwork error: %v", err)
		return nil
	}
	for i := 0; i < len(wires); i++ {
		wire := wires[i].(*SWire)
		if len(self.Properties.IPConfigurations) > 0 {
			network := wire.getNetworkById(self.Properties.IPConfigurations[0].Properties.Subnet.ID)
			if network != nil {
				return network
			}
		}
	}
	return nil
}

func (self *SRegion) GetNetworkInterfaceDetail(interfaceId string) (*SInstanceNic, error) {
	instancenic := SInstanceNic{}
	return &instancenic, self.client.Get(interfaceId, []string{}, &instancenic)
}

func (self *SRegion) GetNetworkInterfaces() ([]SInstanceNic, error) {
	interfaces := []SInstanceNic{}
	err := self.client.ListAll("Microsoft.Network/networkInterfaces", &interfaces)
	if err != nil {
		return nil, err
	}
	result := []SInstanceNic{}
	for i := 0; i < len(interfaces); i++ {
		if interfaces[i].Location == self.Name {
			result = append(result, interfaces[i])
		}
	}
	return result, nil
}

func (self *SRegion) CreateNetworkInterface(nicName string, ipAddr string, subnetId string, secgrpId string) (*SInstanceNic, error) {
	instancenic := SInstanceNic{
		Name:     nicName,
		Location: self.Name,
		Properties: InterfacePropertiesFormat{
			IPConfigurations: []InterfaceIPConfiguration{
				{
					Name: nicName,
					Properties: InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAddress:          ipAddr,
						PrivateIPAddressVersion:   "IPv4",
						PrivateIPAllocationMethod: "Static",
						Subnet: Subnet{
							ID: subnetId,
						},
					},
				},
			},
			NetworkSecurityGroup: &SSecurityGroup{ID: secgrpId},
		},
		Type: "Microsoft.Network/networkInterfaces",
	}

	return &instancenic, self.client.Create(jsonutils.Marshal(&instancenic), &instancenic)
}
