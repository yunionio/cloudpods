package cloudregion

import (
	"time"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var Manager common.IResourceManager[models.SCloudregion]

func init() {
	Manager = NewResourceManager()
}

func NewResourceManager() common.IResourceManager[models.SCloudregion] {
	cm := common.NewCommonResourceManager(
		"cloudregion",
		15*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func NewResourceStore() common.IResourceStore[models.SCloudregion] {
	return common.NewResourceStore[models.SCloudregion](
		models.CloudregionManager,
		compute.Cloudregions,
	)
}
