package zone

import (
	"time"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var Manager common.IResourceManager[models.SZone]

func init() {
	Manager = NewResourceManager()
}

func NewResourceManager() common.IResourceManager[models.SZone] {
	cm := common.NewCommonResourceManager(
		"zone",
		15*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func NewResourceStore() common.IResourceStore[models.SZone] {
	return common.NewResourceStore[models.SZone](
		models.ZoneManager,
		compute.Zones,
	)
}
