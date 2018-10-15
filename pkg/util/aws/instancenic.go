package aws

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SInstanceNic struct {
	instance *SInstance
	ipAddr   string
}

func (self *SInstanceNic) GetIP() string  {
	return ""
}

func (self *SInstanceNic) GetMAC() string {
	return ""
}

func (self *SInstanceNic) GetDriver() string {
	return ""
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork  {
	return nil
}