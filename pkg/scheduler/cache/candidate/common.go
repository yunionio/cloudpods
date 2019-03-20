package candidate

import (
	//"yunion.io/x/log"

	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
)

var (
	VMRunningStatus = sets.NewString(
		models.VM_START_START,
		models.VM_STARTING,
		models.VM_RUNNING,
		models.VM_STOP_FAILED,
		models.VM_BLOCK_STREAM,
		models.VM_UNKNOWN,
		models.VM_BACKUP_STARTING,
	)

	VMCreatingStatus = sets.NewString(
		models.VM_CREATE_NETWORK,
		models.VM_CREATE_DISK,
		models.VM_START_DEPLOY,
		models.VM_DEPLOYING,
		models.VM_BACKUP_CREATING,
		models.VM_DEPLOYING_BACKUP,
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
