package network

import (
	"time"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var Manager common.IResourceManager[models.SNetwork]

func init() {
	Manager = NewResourceManager()
}

func NewResourceManager() common.IResourceManager[models.SNetwork] {
	cm := common.NewCommonResourceManager(
		"network",
		10*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func NewResourceStore() common.IResourceStore[models.SNetwork] {
	return common.NewResourceStore[models.SNetwork](
		models.NetworkManager,
		compute.Networks,
	)
}
