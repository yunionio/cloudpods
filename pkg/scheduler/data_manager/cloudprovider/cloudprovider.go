package cloudprovider

import (
	"time"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var Manager common.IResourceManager[models.SCloudprovider]

func init() {
	Manager = NewResourceManager()
}

func NewResourceManager() common.IResourceManager[models.SCloudprovider] {
	cm := common.NewCommonResourceManager(
		"cloudprovider",
		15*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func NewResourceStore() common.IResourceStore[models.SCloudprovider] {
	return common.NewResourceStore[models.SCloudprovider](
		models.CloudproviderManager,
		compute.Cloudproviders,
	)
}
