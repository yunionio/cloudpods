package google

import (
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SXpnNetwork struct {
	multicloud.SVpc
	multicloud.STagBase

	region *SRegion

	Subnetwork  string
	Network     string
	IpCidrRange string
	StackType   string
	Purpose     string
}

func (network *SXpnNetwork) GetGlobalVpcId() string {
	return getGlobalId(network.Network)
}

func (network *SXpnNetwork) GetId() string {
	return getGlobalId(network.Subnetwork)
}

func (network *SXpnNetwork) GetName() string {
	info := strings.Split(network.GetId(), "/")
	if len(info) == 6 {
		return fmt.Sprintf("%s(%s)", info[5], info[1])
	}
	return network.GetId()
}

func (network *SXpnNetwork) GetGlobalId() string {
	return network.GetId()
}

func (network *SXpnNetwork) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (network *SXpnNetwork) Refresh() error {
	return nil
}

func (network *SXpnNetwork) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (network *SXpnNetwork) GetCreatedAt() time.Time {
	return time.Time{}
}

func (network *SXpnNetwork) GetCidrBlock() string {
	return network.IpCidrRange
}

func (network *SXpnNetwork) IsEmulated() bool {
	return true
}

func (network *SXpnNetwork) GetIsDefault() bool {
	return false
}

func (network *SXpnNetwork) GetRegion() cloudprovider.ICloudRegion {
	return network.region
}

func (network *SXpnNetwork) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (network *SXpnNetwork) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (network *SXpnNetwork) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (network *SXpnNetwork) getWire() *SWire {
	return &SWire{shareVpc: network}
}

func (network *SXpnNetwork) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return []cloudprovider.ICloudWire{network.getWire()}, nil
}

func (network *SXpnNetwork) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if wireId != network.getWire().GetGlobalId() {
		return nil, cloudprovider.ErrNotFound
	}
	return network.getWire(), nil
}

func (client *SGoogleClient) GetXpnNetworks(projectId string) ([]SXpnNetwork, error) {
	res := fmt.Sprintf("projects/%s/aggregated/subnetworks/listUsable", projectId)
	resp, err := client.ecsList(res, nil)
	if err != nil {
		return nil, err
	}
	ret := []SXpnNetwork{}
	err = resp.Unmarshal(&ret, "items")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
