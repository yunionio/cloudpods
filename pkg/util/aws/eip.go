package aws

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type SEipAddress struct {
	region *SRegion

	AllocationId string
	Tags []*ec2.Tag
	InstanceId   string
	AssociationId string
	Domain string
	NetworkInterfaceId string
	NetworkInterfaceOwnerId string
	PrivateIpAddress string
	IpAddress string
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
		Tags: ip.Tags,
		InstanceId: *ip.InstanceId,
		AssociationId: *ip.AssociationId,
		Domain: *ip.Domain,
		NetworkInterfaceId: *ip.NetworkInterfaceId,
		NetworkInterfaceOwnerId:*ip.NetworkInterfaceOwnerId,
		PrivateIpAddress:*ip.PrivateIpAddress,
		IpAddress:*ip.PublicIp,
		})
	}
	
	return eips, len(eips), nil
}