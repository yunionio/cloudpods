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
	//"yunion.io/x/log"

	"yunion.io/x/pkg/util/sets"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
	)

	VMCreatingStatus = sets.NewString(
		computeapi.VM_SCHEDULE,
		computeapi.VM_CREATE_NETWORK,
		computeapi.VM_CREATE_DISK,
		computeapi.VM_START_DEPLOY,
		computeapi.VM_DEPLOYING,
		computeapi.VM_BACKUP_CREATING,
		computeapi.VM_DEPLOYING_BACKUP,
	)
)

func FetchGuestByHostIDs(ids []string) ([]models.SGuest, error) {
	gs := make([]models.SGuest, 0)
	q := models.GuestManager.Query().In("host_id", ids)
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
