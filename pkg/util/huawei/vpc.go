package huawei

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090625.html
type SVpc struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	ID                  string `json:"id"`
	Name                string `json:"name"`
	CIDR                string `json:"cidr"`
	Status              string `json:"status"`
	EnterpriseProjectID string `json:"enterprise_project_id"`
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) getWireByRegionId(regionId string) *SWire {
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

func (self *SVpc) fetchNetworks() error {
	limit := 100
	marker := ""
	networks := make([]SNetwork, 0)
	for {
		parts, count, err := self.region.GetNetwroks(self.ID, limit, marker)
		if err != nil {
			return err
		}

		networks = append(networks, parts...)
		if count <= limit {
			break
		}

		marker = parts[count-1].ID
	}

	if len(networks) == 0 {
		self.iwires = append(self.iwires, &SWire{region: self.region, vpc: self})
		return nil
	}

	for i := 0; i < len(networks); i += 1 {
		wire := self.getWireByRegionId(self.region.GetId())
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}
	return nil
}

// 华为云安全组可以被同region的VPC使用
func (self *SVpc) fetchSecurityGroups() error {
	limit := 100
	marker := ""
	secgroups := make([]SSecurityGroup, 0)
	for {
		// todo： vpc 和 安全组的关联关系还需要进一步确认。
		parts, count, err := self.region.GetSecurityGroups("", limit, marker)
		if err != nil {
			return err
		}

		secgroups = append(secgroups, parts...)
		if count <= limit {
			break
		}

		marker = parts[count-1].ID
	}

	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	// 这里已经填充了vpc。 所以是不是不需要在GetSecurityGroups方法中填充vpc和region了？
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = self
		self.secgroups[i] = &secgroups[i]
	}
	return nil
}

func (self *SVpc) GetId() string {
	return self.ID
}

func (self *SVpc) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SVpc) GetGlobalId() string {
	return self.ID
}

func (self *SVpc) GetStatus() string {
	return models.VPC_STATUS_AVAILABLE
}

func (self *SVpc) Refresh() error {
	new, err := self.region.getVpc(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetIsDefault() bool {
	// 华为云没有default vpc.
	return false
}

func (self *SVpc) GetCidrBlock() string {
	return self.CIDR
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVpc) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SVpc) Delete() error {
	// todo: 确定删除VPC的逻辑
	return self.region.DeleteVpc(self.GetId())
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
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

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpc := SVpc{}
	err := DoGet(self.ecsClient.Vpcs.Get, vpcId, nil, &vpc)
	if strings.Contains(err.Error(), "RouterNotFound") {
		return nil, cloudprovider.ErrNotFound
	}
	vpc.region = self
	return &vpc, err
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	return DoDelete(self.ecsClient.Vpcs.Delete, vpcId, nil, nil)
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090625.html
func (self *SRegion) GetVpcs(limit int, marker string) ([]SVpc, int, error) {
	querys := map[string]string{"limit": "100"}
	if limit > 0 {
		querys["limit"] = strconv.Itoa(limit)
	}

	if len(marker) > 0 {
		querys["marker"] = marker
	}

	vpcs := make([]SVpc, 0)
	err := DoList(self.ecsClient.Vpcs.List, querys, &vpcs)

	for i := range vpcs {
		vpcs[i].region = self
	}
	return vpcs, len(vpcs), err
}
