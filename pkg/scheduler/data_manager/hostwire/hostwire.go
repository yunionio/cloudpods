package hostwire

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var manager common.IResourceManager[models.SHostwire]

func GetManager() common.IResourceManager[models.SHostwire] {
	if manager != nil {
		return manager
	}
	manager = NewResourceManager()
	return manager
}

func NewResourceManager() common.IResourceManager[models.SHostwire] {
	cm := common.NewCommonResourceManager(
		"hostwire",
		10*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func GetId(hostId, wireId string) string {
	return fmt.Sprintf("%s/%s", hostId, wireId)
}

func NewResourceStore() common.IResourceStore[models.SHostwire] {
	return common.NewJointResourceStore(
		models.HostwireManager,
		compute.Hostwires,
		func(o models.SHostwire) string {
			return GetId(o.HostId, o.WireId)
		},
		func(o *jsonutils.JSONDict) string {
			hostId, _ := o.GetString("host_id")
			wireId, _ := o.GetString("wire_id")
			return GetId(hostId, wireId)
		},
	)
}

func GetByHost(hostId string) []models.SHostwire {
	return GetManager().GetStore().GetByPrefix(hostId)
}
