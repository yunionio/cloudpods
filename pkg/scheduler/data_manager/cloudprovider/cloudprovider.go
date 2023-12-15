package cloudprovider

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
)

var manager common.IResourceManager[models.SCloudprovider]

func GetManager() common.IResourceManager[models.SCloudprovider] {
	if manager != nil {
		return manager
	}
	manager = NewResourceManager()
	return manager
}

func NewResourceManager() common.IResourceManager[models.SCloudprovider] {
	cm := common.NewCommonResourceManager(
		"cloudprovider",
		15*time.Minute,
		NewResourceStore(),
	)
	return cm
}

func NewResourceStore() common.IResourceStore[models.SCloudprovider] {
	return common.NewResourceStore[models.SCloudprovider](
		models.CloudproviderManager,
		compute.Cloudproviders,
	).WithOnUpdate(onCloudproviderUpdate)
}

func onCloudproviderUpdate(oldObj *jsonutils.JSONDict, newObj db.IModel) {
	syncStatusKey := "sync_status"
	if !oldObj.Contains(syncStatusKey) {
		return
	}
	// process cloudaccount syncing finished status
	cp := newObj.(*models.SCloudprovider)
	prevStatus, _ := oldObj.GetString(syncStatusKey)
	curStatus := cp.SyncStatus
	if prevStatus == computeapi.CLOUD_PROVIDER_SYNC_STATUS_SYNCING && curStatus == computeapi.CLOUD_PROVIDER_SYNC_STATUS_IDLE {
		if err := onCloudproviderSyncFinished(cp); err != nil {
			log.Infof("onCloudproviderSyncFinished error: %v", err)
		}
	}
}

func onCloudproviderSyncFinished(cp *models.SCloudprovider) error {
	cpHint := fmt.Sprintf("%s/%s", cp.GetId(), cp.GetName())
	hostdIds, err := db.FetchField(models.HostManager, "id", func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", cp.GetId())
	})
	if err != nil {
		return errors.Wrapf(err, "get all hostdIds from cloudprovider %s", cpHint)
	}
	log.Infof("Start reload cloudprovider %s hosts: %v", cpHint, hostdIds)
	if _, err := common.GetCacheManager().ReloadHosts(hostdIds); err != nil {
		return errors.Wrapf(err, "Reload cache hosts of cloudprovider %s", cpHint)
	}
	return nil
}
