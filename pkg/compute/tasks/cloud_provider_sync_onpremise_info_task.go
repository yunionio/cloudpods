package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func syncOnPremiseCloudProviderInfo(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, syncRange *models.SSyncRange) {
	ihosts, err := driver.GetOnPremiseIHosts()
	if err != nil {
		msg := fmt.Sprintf("GetOnPremiseIHosts for provider %s failed %s", provider.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}

	localHosts, remoteHosts, result := models.HostManager.SyncHosts(ctx, task.UserCred, provider, nil, ihosts)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHosts for provider %s result: %s", provider.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)

	for i := 0; i < len(localHosts); i += 1 {
		if len(syncRange.Host) > 0 && !utils.IsInStringArray(localHosts[i].Id, syncRange.Host) {
			continue
		}
		syncHostStorages(ctx, provider, task, &localHosts[i], remoteHosts[i])
		syncHostNics(ctx, provider, task, &localHosts[i], remoteHosts[i])
		syncHostVMs(ctx, provider, task, &localHosts[i], remoteHosts[i], syncRange)
	}
}

func syncHostNics(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localHost *models.SHost, remoteHost cloudprovider.ICloudHost) {
	result := localHost.SyncHostExternalNics(ctx, task.GetUserCred(), remoteHost)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostWires for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.GetUserCred())
	logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
}
