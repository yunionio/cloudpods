package azure

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SObject struct {
	container *SContainer

	cloudprovider.SBaseCloudObject
}

func (o *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return o.container.storageaccount
}
