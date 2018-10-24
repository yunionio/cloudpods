package azure

import (
	"fmt"
	"regexp"

	"yunion.io/x/jsonutils"
)

func (self *SRegion) GetNetworkInterfaceDetail(interfaceId string) (*SInstanceNic, error) {
	instancenic := SInstanceNic{}
	return &instancenic, self.client.Get(interfaceId, &instancenic)
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

func (self *SRegion) isNetworkInstanceNameAvaliable(resourceGroupName, nicName string) (bool, error) {
	nics := []SInstanceNic{}
	err := self.client.ListByTypeWithResourceGroup(resourceGroupName, "Microsoft.Network/networkInterfaces", &nics)
	if err != nil {
		return false, err
	}
	for i := 0; i < len(nics); i++ {
		if nics[i].Name == nicName {
			return false, nil
		}
	}
	return true, nil
}

func getResourceGroupNameByID(id string) string {
	reg := regexp.MustCompile("/resourceGroups/(.+)/providers/")
	_resourceGroup := reg.FindStringSubmatch(id)
	if len(_resourceGroup) == 2 {
		return _resourceGroup[1]
	}
	return ""
}

func (self *SRegion) CreateNetworkInterface(nicName string, ipAddr string, subnetId string, secgrpId string) (*SInstanceNic, error) {
	secgroup, err := self.GetSecurityGroupDetails(secgrpId)
	if err != nil {
		return nil, err
	}
	secgroup.Properties.ProvisioningState = ""

	resourceGroupName := getResourceGroupNameByID(subnetId)
	nicNameBase := nicName
	for i := 0; i < 5; i++ {
		ok, err := self.isNetworkInstanceNameAvaliable(resourceGroupName, nicName)
		if err != nil {
			return nil, err
		}
		if ok {
			break
		}
		nicName = fmt.Sprintf("%s-%d", nicNameBase, i)
	}

	instancenic := SInstanceNic{
		Name:     nicName,
		Location: self.Name,
		Properties: InterfacePropertiesFormat{
			IPConfigurations: []InterfaceIPConfiguration{
				InterfaceIPConfiguration{
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
			NetworkSecurityGroup: secgroup,
		},
		Type: "Microsoft.Network/networkInterfaces",
	}

	return &instancenic, self.client.Create(jsonutils.Marshal(&instancenic), &instancenic)
}
