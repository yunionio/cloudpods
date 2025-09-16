// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package netinterface

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

func GetId(hostId, wireId, mac string, vlanId int) string {
	return fmt.Sprintf("%s/%s/%s/%d", hostId, wireId, mac, vlanId)
}

func NewResourceStore() common.IResourceStore[models.SNetInterface] {
	return common.NewJointResourceStore(
		models.NetInterfaceManager,
		compute.Networkinterfaces,
		func(o models.SNetInterface) string {
			return GetId(o.BaremetalId, o.WireId, o.Mac, o.VlanId)
		},
		func(o *jsonutils.JSONDict) string {
			hostId, _ := o.GetString("baremetal_id")
			wireId, _ := o.GetString("wire_id")
			mac, _ := o.GetString("mac")
			vlan, _ := o.Int("vlan_id")
			return GetId(hostId, wireId, mac, int(vlan))
		},
		func(man db.IModelManager, id string, obj *jsonutils.JSONDict) (db.IModel, error) {
			ids := strings.Split(id, "/")
			if len(ids) != 3 {
				return nil, errors.Errorf("Invalid id: %q", id)
			}
			q := man.Query()
			hostId := ids[0]
			wireId := ids[1]
			mac := ids[2]
			q = q.Equals("host_id", hostId).Equals("wire_id", wireId).Equals("mac", mac)
			objs, err := db.FetchIModelObjects(man, q)
			errHint := fmt.Sprintf("hostId %q, wireId %q, mac %q", hostId, wireId, mac)
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

func GetByHost(hostId string) []models.SNetInterface {
	return GetManager().GetStore().GetByPrefix(hostId)
}
