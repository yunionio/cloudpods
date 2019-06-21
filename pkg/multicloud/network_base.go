package multicloud

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SNetworkBase struct {
}

func (self *SNetworkBase) GetReservedIps() ([]cloudprovider.SReservedIp, error) {
	return nil, fmt.Errorf("Not Implemented GetReservedIps")
}
