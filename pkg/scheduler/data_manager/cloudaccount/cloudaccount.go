package cloudaccount

import (
	"time"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var Manager common.IResourceManager[models.SCloudaccount]

func init() {
	Manager = NewResourceManager()
}

func NewResourceManager() common.IResourceManager[models.SCloudaccount] {
	cm := common.NewCommonResourceManager(
		"cloudaccount",
		15*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func NewResourceStore() common.IResourceStore[models.SCloudaccount] {
	return common.NewResourceStore[models.SCloudaccount](
		models.CloudaccountManager,
		compute.Cloudaccounts,
	)
}
