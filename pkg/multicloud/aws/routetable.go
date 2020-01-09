package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRouteTable struct {
	region *SRegion
	vpc    *SVpc

	Associations    []Association `json:"Associations"`
	PropagatingVgws []string      `json:"PropagatingVgws"`
	RouteTableID    string        `json:"RouteTableId"`
	Routes          []SRoute      `json:"Routes"`
	VpcID           string        `json:"VpcId"`
	OwnerID         string        `json:"OwnerId"`
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
	return ""
}

func (self *SRouteTable) GetGlobalId() string {
	return self.GetId()
}

func (self *SRouteTable) GetStatus() string {
	return ""
}

func (self *SRouteTable) Refresh() error {
	ret, err := self.region.GetRouteTable(self.GetId())
	if err != nil {
		return errors.Wrap(err, "SRouteTable.Refresh.GetRouteTable")
	}

	err = jsonutils.Update(self, ret)
	if err != nil {
		return errors.Wrap(err, "SRouteTable.Refresh.Update")
	}

	return nil
}

func (self *SRouteTable) IsEmulated() bool {
	return false
}

func (self *SRouteTable) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRouteTable) GetDescription() string {
	return ""
}

func (self *SRouteTable) GetRegionId() string {
	return self.region.GetId()
}

func (self *SRouteTable) GetVpcId() string {
	return self.VpcID
}

func (self *SRouteTable) GetType() string {
	return api.ROUTE_TABLE_TYPE_VPC
}

func (self *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	iroutes := make([]cloudprovider.ICloudRoute, len(self.Routes))
	for i := range self.Routes {
		self.Routes[i].routetable = self
		iroutes[i] = &self.Routes[i]
	}

	return iroutes, nil
}

func (self *SRegion) GetRouteTables(vpcId string, mainRouteOnly bool) ([]SRouteTable, error) {
	input := &ec2.DescribeRouteTablesInput{}
	filters := make([]*ec2.Filter, 0)
	filters = AppendSingleValueFilter(filters, "vpc-id", vpcId)
	if mainRouteOnly {
		filters = AppendSingleValueFilter(filters, "association.main", "true")
	}

	input.SetFilters(filters)

	ret, err := self.ec2Client.DescribeRouteTables(input)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.DescribeRouteTables")
	}

	routeTables := make([]SRouteTable, len(ret.RouteTables))
	err = unmarshalAwsOutput(ret, "RouteTables", routeTables)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.unmarshalAwsOutput")
	}

	for i := range routeTables {
		routeTables[i].region = self
	}

	return routeTables, nil
}

func (self *SRegion) GetRouteTablesByNetworkId(netId string) ([]SRouteTable, error) {
	input := &ec2.DescribeRouteTablesInput{}
	filter := &ec2.Filter{}
	filter.SetName("association.subnet-id")
	filter.SetValues([]*string{&netId})
	input.SetFilters([]*ec2.Filter{filter})

	ret, err := self.ec2Client.DescribeRouteTables(input)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.DescribeRouteTables")
	}

	routeTables := make([]SRouteTable, len(ret.RouteTables))
	err = unmarshalAwsOutput(ret, "RouteTables", routeTables)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.unmarshalAwsOutput")
	}

	for i := range routeTables {
		routeTables[i].region = self
	}

	return routeTables, nil
}

func (self *SRegion) GetRouteTable(id string) (*SRouteTable, error) {
	input := &ec2.DescribeRouteTablesInput{}
	input.RouteTableIds = []*string{&id}

	ret, err := self.ec2Client.DescribeRouteTables(input)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.DescribeRouteTables")
	}

	routeTables := make([]SRouteTable, len(ret.RouteTables))
	err = unmarshalAwsOutput(ret, "RouteTables", routeTables)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.unmarshalAwsOutput")
	}

	if len(routeTables) == 1 {
		routeTables[0].region = self
		return &routeTables[0], nil
	} else if len(routeTables) == 0 {
		return nil, errors.ErrNotFound
	} else {
		return nil, errors.ErrDuplicateId
	}
}
