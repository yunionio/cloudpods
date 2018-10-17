package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"time"
)

type SEipAddress struct {
	region *SRegion

	AllocationId            string
	Bandwidth            int
	Tags                    STags
	Status    string
	InstanceId              string
	AssociationId           string
	Domain                  string
	NetworkInterfaceId      string
	NetworkInterfaceOwnerId string
	PrivateIpAddress        string
	IpAddress               string
}

func (self *SEipAddress) GetId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetName() string {
	return self.IpAddress
}

func (self *SEipAddress) GetGlobalId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetStatus() string {
	switch self.Status {
	default:
		return models.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEip(self.AllocationId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEipAddress) IsEmulated() bool {
	if self.AllocationId == self.InstanceId {
		return true
	}

	return false
}

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetIpAddr() string {
	return self.IpAddress
}

func (self *SEipAddress) GetMode() string {
	if self.InstanceId == self.AllocationId {
		return models.EIP_MODE_INSTANCE_PUBLICIP
	} else {
		return models.EIP_MODE_STANDALONE_EIP
	}
}

func (self *SEipAddress) GetAssociationType() string {
	// todo : ?
	return "server"
}

func (self *SEipAddress) GetAssociationExternalId() string {
	return self.InstanceId
}

func (self *SEipAddress) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SEipAddress) GetInternetChargeType() string {
	// todo : implement me
	return models.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEipAddress) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.AllocationId)
}

func (self *SEipAddress) Associate(instanceId string) error {
	err := self.region.AssociateEip(self.AllocationId, instanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.AllocationId, self.InstanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.AllocationId, bw)
}

func (self *SRegion) GetEips(eipId string) ([]SEipAddress, int, error) {
	params := ec2.DescribeAddressesInput{}
	if len(eipId) > 0 {
		params.SetAllocationIds([]*string{&eipId})
	}

	res, err := self.ec2Client.DescribeAddresses(&params)
	if err != nil {
		log.Errorf("DescribeEipAddresses fail %s", err)
		return nil, 0, err
	}

	eips := make([]SEipAddress, 0)
	for _, ip := range res.Addresses {
		eips = append(eips, SEipAddress{region: self, AllocationId: *ip.AllocationId,
			Tags:                    STags{},
			InstanceId:              *ip.InstanceId,
			AssociationId:           *ip.AssociationId,
			Domain:                  *ip.Domain,
			NetworkInterfaceId:      *ip.NetworkInterfaceId,
			NetworkInterfaceOwnerId: *ip.NetworkInterfaceOwnerId,
			PrivateIpAddress:        *ip.PrivateIpAddress,
			IpAddress:               *ip.PublicIp,
		})
	}
	return eips, len(eips), nil
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	return nil, nil
}

func (region *SRegion) AllocateEIP(bwMbps int) (*SEipAddress, error) {
	return nil, nil
}

func (self *SRegion) CreateEIP(name string, bwMbps int, chargeType string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.ec2Client.AllocateAddress(&ec2.AllocateAddressInput{})
	if err != nil {
		log.Errorf("AllocateEipAddress fail %s", err)
		return nil, err
	}

	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetIEipById(*eip.AllocationId)
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	return nil
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	return nil
}

func (region *SRegion) DissociateEip(eipId string, instanceId string) error {
	return nil
}

func (region *SRegion) UpdateEipBandwidth(eipId string, bw int) error {
	return nil
}