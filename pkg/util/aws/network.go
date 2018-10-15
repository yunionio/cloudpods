package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type SNetwork struct {
	wire *SWire

	AvailableIpAddressCount int
	CidrBlock               string
	CreationTime            time.Time
	Description             string
	IsDefault               bool
	Status                  string
	NetworkId               string
	NetworkName             string
	VpcId                   string
	ZoneId                  string
}

func (self *SNetwork) GetId() string {
	panic("implement me")
}

func (self *SNetwork) GetName() string {
	panic("implement me")
}

func (self *SNetwork) GetGlobalId() string {
	panic("implement me")
}

func (self *SNetwork) GetStatus() string {
	panic("implement me")
}

func (self *SNetwork) Refresh() error {
	panic("implement me")
}

func (self *SNetwork) IsEmulated() bool {
	panic("implement me")
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	panic("implement me")
}

func (self *SNetwork) GetIpStart() string {
	panic("implement me")
}

func (self *SNetwork) GetIpEnd() string {
	panic("implement me")
}

func (self *SNetwork) GetIpMask() int8 {
	panic("implement me")
}

func (self *SNetwork) GetGateway() string {
	panic("implement me")
}

func (self *SNetwork) GetServerType() string {
	panic("implement me")
}

func (self *SNetwork) GetIsPublic() bool {
	panic("implement me")
}

func (self *SNetwork) Delete() error {
	panic("implement me")
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	panic("implement me")
}

func (self *SRegion) createNetwork(zoneId string, vpcId string, name string, cidr string, desc string) (string, error)  {
	return "", nil
}

func (self *SRegion) getNetwork(vswitchId string) (*SNetwork, error) {
	return nil, nil
}

func (self *SRegion) deleteNetwork(vswitchId string) error {
	return nil
}

func (self *SRegion) GetNetwroks(ids []string, vpcId string) ([]SNetwork, int, error) {
	params := &ec2.DescribeSubnetsInput{}
	if len(ids) > 0 {
		_ids := make([]*string, len(ids))
		for _, id := range ids{
			_ids = append(_ids, &id)
		}
		params.SetSubnetIds(_ids)
	}

	if len(vpcId) > 0 {
		filters := make([]*ec2.Filter, 1)
		vpcFilter := &ec2.Filter{}
		vpcFilter.SetName("vpc-id")
		vpcFilter.SetValues([]*string{&vpcId})
		filters = append(filters, vpcFilter)
		params.SetFilters(filters)
	}

	items, err := self.ec2Client.DescribeSubnets(params)
	if err != nil {
		return nil, 0, err
	}

	subnets := make([]SNetwork, len(items.Subnets))
	for _, item := range items.Subnets {
		subnet := SNetwork{}
		subnet.CidrBlock = *item.CidrBlock
		subnet.VpcId = *item.VpcId
		subnet.Status = *item.State
		subnet.ZoneId = *item.AvailabilityZone
		subnet.IsDefault = *item.DefaultForAz
		subnet.NetworkId = *item.SubnetId
		subnet.NetworkName = *item.SubnetId
		subnets = append(subnets, subnet)
	}

	return subnets, len(subnets), nil
}
