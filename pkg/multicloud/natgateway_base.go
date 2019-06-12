package multicloud

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SNatGatewayBase struct {
	SResourceBase
	SBillingBase
}

func (nat *SNatGatewayBase) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	return nil, fmt.Errorf("Not Implemented GetIEips")
}

func (nat *SNatGatewayBase) GetINatDTables() ([]cloudprovider.ICloudNatDTable, error) {
	return nil, fmt.Errorf("Not Implemented GetINatDTables")
}

func (nat *SNatGatewayBase) GetINatSTables() ([]cloudprovider.ICloudNatSTable, error) {
	return nil, fmt.Errorf("Not Implemented GetINatSTables")
}
