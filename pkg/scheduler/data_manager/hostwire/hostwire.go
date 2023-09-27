package hostwire

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
		func(man db.IModelManager, id string, obj *jsonutils.JSONDict) (db.IModel, error) {
			ids := strings.Split(id, "/")
			if len(ids) != 2 {
				return nil, errors.Errorf("Invalid id: %q", id)
			}
			q := man.Query()
			hostId := ids[0]
			wireId := ids[1]
			q = q.Equals("host_id", hostId).Equals("wire_id", wireId)
			objs, err := db.FetchIModelObjects(man, q)
			errHint := fmt.Sprintf("hostId %q, wireId %q", hostId, wireId)
			if err != nil {
				return nil, errors.Wrapf(err, "db.FetchIModelObjects by %s", errHint)
			}
			if len(objs) != 1 {
				return nil, errors.Errorf("Found %d objects by %s", len(objs), errHint)
			}
			return objs[0], nil
		},
	)
}

func GetByHost(hostId string) []models.SHostwire {
	return GetManager().GetStore().GetByPrefix(hostId)
}
