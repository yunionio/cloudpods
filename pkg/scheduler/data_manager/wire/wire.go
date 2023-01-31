package wire

import (
	"time"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var Manager common.IResourceManager[models.SWire]

func init() {
	Manager = NewResourceManager()
}

func NewResourceManager() common.IResourceManager[models.SWire] {
	cm := common.NewCommonResourceManager(
		"wire",
		10*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func NewResourceStore() common.IResourceStore[models.SWire] {
	return common.NewResourceStore[models.SWire](
		models.WireManager,
		compute.Wires,
	)
}
