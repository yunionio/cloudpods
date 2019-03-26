package ucloud

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SVPC struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	CreateTime  int64         `json:"CreateTime"`
	Name        string        `json:"Name"`
	Network     []string      `json:"Network"`
	NetworkInfo []NetworkInfo `json:"NetworkInfo"`
	SubnetCount int           `json:"SubnetCount"`
	Tag         string        `json:"Tag"`
	UpdateTime  int64         `json:"UpdateTime"`
	VPCID       string        `json:"VPCId"`
}

type NetworkInfo struct {
	Network     string `json:"Network"`
	SubnetCount int    `json:"SubnetCount"`
}

func (self *SVPC) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVPC) GetId() string {
	return self.VPCID
}

func (self *SVPC) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.VPCID
}

func (self *SVPC) GetGlobalId() string {
	return self.GetId()
}

func (self *SVPC) GetStatus() string {
	return models.VPC_STATUS_AVAILABLE
}

func (self *SVPC) Refresh() error {
	new, err := self.region.getVpc(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVPC) IsEmulated() bool {
	return false
}

func (self *SVPC) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SVPC) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVPC) GetIsDefault() bool {
	return false
}

func (self *SVPC) GetCidrBlock() string {
	return strings.Join(self.Network, ",")
}

func (self *SVPC) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

// 由于Ucloud 安全组和vpc没有直接关联，这里是返回同一个项目下的防火墙列表，会导致重复同步的问题。
// https://docs.ucloud.cn/api/unet-api/grant_firewall
func (self *SVPC) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}

func (self *SVPC) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVPC) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SVPC) Delete() error {
	return self.region.DeleteVpc(self.GetId())
}

func (self *SVPC) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(self.iwires); i += 1 {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVPC) fetchNetworks() error {
	networks, err := self.region.GetNetworks(self.GetId())
	if err != nil {
		return err
	}

	for i := 0; i < len(networks); i += 1 {
		wire := self.getWireByRegionId(self.region.GetId())
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}

	return nil
}

func (self *SVPC) getWireByRegionId(regionId string) *SWire {
	if len(regionId) == 0 {
		return nil
	}

	for i := 0; i < len(self.iwires); i++ {
		wire := self.iwires[i].(*SWire)

		if wire.region.GetId() == regionId {
			return wire
		}
	}

	return nil
}

func (self *SRegion) getVpc(vpcId string) (*SVPC, error) {
	vpcs, err := self.GetVpcs(vpcId)
	if err != nil {
		return nil, err
	}

	if len(vpcs) == 1 {
		return &vpcs[0], nil
	} else if len(vpcs) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, fmt.Errorf("getVpc %s %d found", vpcId, len(vpcs))
	}
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	return cloudprovider.ErrNotImplemented
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090625.html
func (self *SRegion) GetVpcs(vpcId string) ([]SVPC, error) {
	vpcs := make([]SVPC, 0)
	params := NewUcloudParams()
	if len(vpcId) > 0 {
		params.Set("VPCIds.0", vpcId)
	}

	err := self.DoListAll("DescribeVPC", params, &vpcs)
	return vpcs, err
}

func (self *SRegion) GetNetworks(vpcId string) ([]SNetwork, error) {
	params := NewUcloudParams()
	if len(vpcId) == 0 {
		params.Set("VPCId", vpcId)
	}

	networks := make([]SNetwork, 0)
	err := self.DoAction("DescribeSubnet", params, &networks)
	return networks, err
}

// UCLOUD 同一个项目共用安全组（防火墙）
func (self *SVPC) fetchSecurityGroups() error {
	secgroups, err := self.region.GetSecurityGroups("", "")
	if err != nil {
		return err
	}

	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = self
		secgroups[i].region = self.region
		self.secgroups[i] = &secgroups[i]
	}
	return nil
}
