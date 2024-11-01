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

package candidate

import (
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

var (
	VMRunningStatus = sets.NewString(
		computeapi.VM_START_START,
		computeapi.VM_STARTING,
		computeapi.VM_RUNNING,
		computeapi.VM_STOP_FAILED,
		computeapi.VM_BLOCK_STREAM,
		computeapi.VM_UNKNOWN,
		computeapi.VM_BACKUP_STARTING,
		computeapi.POD_STATUS_CONTAINER_EXITED,
		computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF,
		computeapi.POD_STATUS_STARTING_CONTAINER,
		computeapi.POD_STATUS_STOP_CONTAINER_FAILED,
		computeapi.POD_STATUS_STOPPING_CONTAINER,
	)

	VMCreatingStatus = sets.NewString(
		computeapi.VM_SCHEDULE,
		computeapi.VM_CREATE_NETWORK,
		computeapi.VM_CREATE_DISK,
		computeapi.VM_START_DEPLOY,
		computeapi.VM_DEPLOYING,
		computeapi.VM_BACKUP_CREATING,
		computeapi.VM_DEPLOYING_BACKUP,
		computeapi.POD_STATUS_CREATING_CONTAINER,
	)

	VMStoppedStatus = sets.NewString(
		computeapi.VM_READY,
		computeapi.VM_START_FAILED,
		computeapi.VM_SCHEDULE_FAILED,
		computeapi.VM_NETWORK_FAILED,
		computeapi.VM_CREATE_FAILED,
		computeapi.VM_DISK_FAILED,
		computeapi.POD_STATUS_START_CONTAINER_FAILED,
		computeapi.POD_STATUS_CREATE_CONTAINER_FAILED,
	)
)

func GetHostIds(hosts []models.SHost) []string {
	ids := make([]string, len(hosts))
	for i, host := range hosts {
		ids[i] = host.GetId()
	}
	return ids
}

func FetchGuestByHostIDsQuery(idsQuery sqlchemy.IQuery) ([]models.SGuest, error) {
	gs := make([]models.SGuest, 0)

	q := models.GuestManager.Query()
	hostIdsSubq := idsQuery.SubQuery()
	q = q.Join(hostIdsSubq, sqlchemy.Equals(q.Field("host_id"), hostIdsSubq.Field("id")))
	err := db.FetchModelObjects(models.GuestManager, q, &gs)
	return gs, err
}

func IsGuestRunning(g models.SGuest) bool {
	return VMRunningStatus.Has(g.Status)
}

func IsGuestCreating(g models.SGuest) bool {
	return VMCreatingStatus.Has(g.Status)
}

func IsGuestPendingDelete(g models.SGuest) bool {
	return g.PendingDeleted
}

func IsGuestStoppedStatus(g models.SGuest) bool {
	return VMStoppedStatus.Has(g.Status)
}

func ToDict[O lockman.ILockedObject](objs []O) map[string]*O {
	ret := make(map[string]*O, 0)
	for _, obj := range objs {
		tmpObj := obj
		objPtr := &tmpObj
		ret[obj.GetId()] = objPtr
	}
	return ret
}
