package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	return self.NetworkId
}

func (self *SNetwork) GetName() string {
	if len(self.NetworkName) == 0 {
		return self.NetworkId
	}

	return self.NetworkName
}

func (self *SNetwork) GetGlobalId() string {
	return self.NetworkId
}

func (self *SNetwork) GetStatus() string {
	return strings.ToLower(self.Status)
}

func (self *SNetwork) Refresh() error {
	log.Debugf("network refresh %s", self.NetworkId)
	new, err := self.wire.zone.region.getNetwork(self.NetworkId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetServerType() string {
	return models.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) Delete() error {
	return self.wire.zone.region.deleteNetwork(self.NetworkId)
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) createNetwork(zoneId string, vpcId string, name string, cidr string, desc string) (string, error) {
	params := &ec2.CreateSubnetInput{}
	params.SetAvailabilityZone(zoneId)
	params.SetVpcId(vpcId)
	params.SetCidrBlock(cidr)

	ret, err := self.ec2Client.CreateSubnet(params)
	if err != nil {
		return "", err
	} else {
		paramsTags := &ec2.CreateTagsInput{}
		tagspec := TagSpec{ResourceType: "subnet"}
		tagspec.SetNameTag(name)
		tagspec.SetDescTag(desc)
		ec2Tag, _ := tagspec.GetTagSpecifications()
		paramsTags.SetResources([]*string{ret.Subnet.SubnetId})
		paramsTags.SetTags(ec2Tag.Tags)
		_, err := self.ec2Client.CreateTags(paramsTags)
		if err != nil {
			log.Infof("createNetwork write tags failed:%s", err)
		}
		return *ret.Subnet.SubnetId, nil
	}
}

func (self *SRegion) getNetwork(networkId string) (*SNetwork, error) {
	if len(networkId) == 0 {
		return nil, fmt.Errorf("GetNetwork networkId should not be empty.")
	}
	networks, total, err := self.GetNetwroks([]string{networkId}, "", 0, 0)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &networks[0], nil
}

func (self *SRegion) deleteNetwork(networkId string) error {
	params := &ec2.DeleteSubnetInput{}
	params.SetSubnetId(networkId)
	_, err := self.ec2Client.DeleteSubnet(params)
	return err
}

func (self *SRegion) GetNetwroks(ids []string, vpcId string, limit int, offset int) ([]SNetwork, int, error) {
	params := &ec2.DescribeSubnetsInput{}
	if len(ids) > 0 {
		_ids := make([]*string, len(ids))
		for _, id := range ids {
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

	ret, err := self.ec2Client.DescribeSubnets(params)
	if err != nil {
		return nil, 0, err
	}

	subnets := []SNetwork{}
	for i := range ret.Subnets {
		item := ret.Subnets[i]
		if err := FillZero(item); err != nil {
			return nil, 0, err
		}

		tagspec := TagSpec{ResourceType: "subnet"}
		tagspec.LoadingEc2Tags(item.Tags)

		subnet := SNetwork{}
		subnet.CidrBlock = *item.CidrBlock
		subnet.VpcId = *item.VpcId
		subnet.Status = *item.State
		subnet.ZoneId = *item.AvailabilityZone
		subnet.IsDefault = *item.DefaultForAz
		subnet.NetworkId = *item.SubnetId
		subnet.NetworkName = tagspec.GetNameTag()
		subnets = append(subnets, subnet)
	}
	return subnets, len(subnets), nil
}
