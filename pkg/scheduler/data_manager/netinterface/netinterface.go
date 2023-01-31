package netinterface

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var manager common.IResourceManager[models.SNetInterface]

func GetManager() common.IResourceManager[models.SNetInterface] {
	if manager != nil {
		return manager
	}
	manager = NewResourceManager()
	return manager
}

func NewResourceManager() common.IResourceManager[models.SNetInterface] {
	cm := common.NewCommonResourceManager(
		"netinterface",
		10*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func GetId(hostId, wireId, mac string) string {
	return fmt.Sprintf("%s/%s/%s", hostId, wireId, mac)
}

func NewResourceStore() common.IResourceStore[models.SNetInterface] {
	return common.NewJointResourceStore(
		models.NetInterfaceManager,
		compute.Networkinterfaces,
		func(o models.SNetInterface) string {
			return GetId(o.BaremetalId, o.WireId, o.Mac)
		},
		func(o *jsonutils.JSONDict) string {
			hostId, _ := o.GetString("host_id")
			wireId, _ := o.GetString("wire_id")
			mac, _ := o.GetString("mac")
			return GetId(hostId, wireId, mac)
		},
	)
}

func GetByHost(hostId string) []models.SNetInterface {
	return GetManager().GetStore().GetByPrefix(hostId)
}
