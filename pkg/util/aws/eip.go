package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SEipAddress struct {
	region *SRegion

	AllocationId            string
	Tags                    STags
	InstanceId              string
	AssociationId           string
	Domain                  string
	NetworkInterfaceId      string
	NetworkInterfaceOwnerId string
	PrivateIpAddress        string
	IpAddress               string
}

func (self *SEipAddress) GetId() string {
	panic("implement me")
}

func (self *SEipAddress) GetName() string {
	panic("implement me")
}

func (self *SEipAddress) GetGlobalId() string {
	panic("implement me")
}

func (self *SEipAddress) GetStatus() string {
	panic("implement me")
}

func (self *SEipAddress) Refresh() error {
	panic("implement me")
}

func (self *SEipAddress) IsEmulated() bool {
	panic("implement me")
}

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SEipAddress) GetIpAddr() string {
	panic("implement me")
}

func (self *SEipAddress) GetMode() string {
	panic("implement me")
}

func (self *SEipAddress) GetAssociationType() string {
	panic("implement me")
}

func (self *SEipAddress) GetAssociationExternalId() string {
	panic("implement me")
}

func (self *SEipAddress) GetBandwidth() int {
	panic("implement me")
}

func (self *SEipAddress) GetInternetChargeType() string {
	panic("implement me")
}

func (self *SEipAddress) GetManagerId() string {
	panic("implement me")
}

func (self *SEipAddress) Delete() error {
	panic("implement me")
}

func (self *SEipAddress) Associate(instanceId string) error {
	panic("implement me")
}

func (self *SEipAddress) Dissociate() error {
	panic("implement me")
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	panic("implement me")
}

func (region *SRegion) GetEips(eipId string) ([]SEipAddress, int, error) {
	params := ec2.DescribeAddressesInput{}
	if len(eipId) > 0 {
		params.AllocationIds = []*string{&eipId}
	}

	res, err := region.ec2Client.DescribeAddresses(&params)
	if err != nil {
		log.Errorf("DescribeEipAddresses fail %s", err)
		return nil, 0, err
	}

	eips := make([]SEipAddress, 0)
	for _, ip := range res.Addresses {
		eips = append(eips, SEipAddress{region: region, AllocationId: *ip.AllocationId,
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