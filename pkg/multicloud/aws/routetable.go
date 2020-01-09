package aws

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRouteTable struct {
	Associations    []Association `json:"Associations"`
	PropagatingVgws []string      `json:"PropagatingVgws"`
	RouteTableID    string        `json:"RouteTableId"`
	Routes          []SRoute      `json:"Routes"`
	Tags            map[string]string
	VpcID           string `json:"VpcId"`
	OwnerID         string `json:"OwnerId"`
}

type Association struct {
	Main                    bool    `json:"Main"`
	RouteTableAssociationID string  `json:"RouteTableAssociationId"`
	RouteTableID            string  `json:"RouteTableId"`
	SubnetID                *string `json:"SubnetId,omitempty"`
}

func (self *SRouteTable) GetId() string {
	return self.RouteTableID
}

func (self *SRouteTable) GetName() string {
	panic("implement me")
}

func (self *SRouteTable) GetGlobalId() string {
	panic("implement me")
}

func (self *SRouteTable) GetStatus() string {
	panic("implement me")
}

func (self *SRouteTable) Refresh() error {
	panic("implement me")
}

func (self *SRouteTable) IsEmulated() bool {
	panic("implement me")
}

func (self *SRouteTable) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SRouteTable) GetDescription() string {
	panic("implement me")
}

func (self *SRouteTable) GetRegionId() string {
	panic("implement me")
}

func (self *SRouteTable) GetVpcId() string {
	panic("implement me")
}

func (self *SRouteTable) GetType() string {
	panic("implement me")
}

func (self *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	panic("implement me")
}
