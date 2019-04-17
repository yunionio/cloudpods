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
