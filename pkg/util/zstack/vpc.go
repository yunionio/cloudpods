package zstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SVpc struct {
	region *SRegion

	iwires []cloudprovider.ICloudWire
}

func (vpc *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (vpc *SVpc) GetId() string {
	return fmt.Sprintf("%s/vpc", vpc.region.GetGlobalId())
}

func (vpc *SVpc) GetName() string {
	return DEFAULT_VPC_NAME
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.GetId()
}

func (vpc *SVpc) IsEmulated() bool {
	return true
}

func (vpc *SVpc) GetIsDefault() bool {
	return true
}

func (vpc *SVpc) GetCidrBlock() string {
	return ""
}

func (vpc *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (vpc *SVpc) Refresh() error {
	return nil
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if vpc.iwires == nil || len(vpc.iwires) == 0 {
		vpc.iwires = []cloudprovider.ICloudWire{}
		wires, err := vpc.region.GetWires("", "", "")
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(wires); i++ {
			wires[i].vpc = vpc
			vpc.iwires = append(vpc.iwires, &wires[i])
		}
	}
	return vpc.iwires, nil
}

func (vpc *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wire, err := vpc.region.GetWire(wireId)
	if err != nil {
		return nil, err
	}
	wire.vpc = vpc
	return wire, nil
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := vpc.region.GetSecurityGroups("", "")
	if err != nil {
		return nil, err
	}
	isecgroups := []cloudprovider.ICloudSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		isecgroups = append(isecgroups, &secgroups[i])
	}
	return isecgroups, nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (vpc *SVpc) GetManagerId() string {
	return vpc.region.client.providerID
}

func (vpc *SVpc) Delete() error {
	return nil
}
