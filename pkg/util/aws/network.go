package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SNetwork struct {
	wire *SWire

	AvailableIpAddressCount int
	CidrBlock               string
	CreationTime            time.Time
	Description             string
	IsDefault               bool
	Status                  string
	VSwitchId               string
	VSwitchName             string
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

