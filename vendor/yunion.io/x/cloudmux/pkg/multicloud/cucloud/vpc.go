package cucloud

import (
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
)

type SVpc struct {
	multicloud.SVpc
	multicloud.STagBase

	region *SRegion

	BandWidth         string
	RouterUUID        string
	VpcName           string
	Flag              string
	CloudRegionId     string
	Description       string
	RouterTableNum    int
	InstanceId        string
	RouterName        string
	RouterID          string
	VrouterType       string
	VpcId             string
	DnatId            string
	VpcType           string
	Cidr              string
	ResourceGroupId   string
	CloudRegionName   string
	SubNetNum         int
	ResourceGroupName string
	IsdefaultNetwork  string
	SubNetworkNum     int
	CreateTime        string
	RouterNum         int
	IsDefaultNetwork  bool
	Status            string
}

func (vpc *SVpc) GetId() string {
	return vpc.VpcId
}

func (vpc *SVpc) GetName() string {
	return vpc.VpcName
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.VpcId
}

func (vpc *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (vpc *SVpc) Refresh() error {
	res, err := vpc.region.GetVpc(vpc.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(vpc, res)
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (vpc *SVpc) GetIsDefault() bool {
	return vpc.IsDefaultNetwork
}

func (vpc *SVpc) GetCidrBlock() string {
	return vpc.Cidr
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	zones, err := vpc.region.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range zones {
		zones[i].region = vpc.region
		wire := &SWire{vpc: vpc, zone: &zones[i]}
		ret = append(ret, wire)
	}
	return ret, nil
}

func (vpc *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wires, err := vpc.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == id {
			return wires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetVpcs(id string) ([]SVpc, error) {
	params := url.Values{}
	params.Set("cloudRegionCode", region.CloudRegionCode)
	if len(id) > 0 {
		params.Set("vpcId", id)
	}
	resp, err := region.list("/instance/v1/product/vpcs", params)
	if err != nil {
		return nil, err
	}
	ret := []SVpc{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	vpc, err := region.GetVpcs(vpcId)
	if err != nil {
		return nil, err
	}
	for i := range vpc {
		vpc[i].region = region
		if vpc[i].VpcId == vpcId {
			return &vpc[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
