package tasks

import (
	"context"
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/utils"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type CloudProviderSyncInfoTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudProviderSyncInfoTask{})
}

func taskFail(ctx context.Context, task *CloudProviderSyncInfoTask, provider *models.SCloudprovider, reason string) {
	provider.SetStatus(task.UserCred, models.CLOUD_PROVIDER_DISCONNECTED, reason)
	task.SetStageFailed(ctx, reason)
}

func (self *CloudProviderSyncInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)
	provider.MarkStartSync(self.UserCred)
	// do sync

	log.Infof("Start sync cloud provider status ...")
	driver, err := provider.GetDriver()
	if err != nil {
		reason := fmt.Sprintf("Invalid cloud provider %s", err)
		taskFail(ctx, self, provider, reason)
		return
	}

	sysinfo, err := driver.GetSysInfo()
	if err != nil {
		reason := fmt.Sprintf("provider get sysinfo error %s", err)
		taskFail(ctx, self, provider, reason)
		return
	} else {
		provider.SaveSysInfo(sysinfo)
	}

	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRange := models.SSyncRange{}
		err = syncRangeJson.Unmarshal(&syncRange)
		if err == nil {
			syncCloudProviderInfo(ctx, provider, self, driver, &syncRange)
		}
	}

	provider.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
	self.SetStageComplete(ctx, nil)
}

func logSyncFailed(provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, reason string) {
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, reason, task.UserCred)
}

func syncCloudProviderInfo(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, syncRange *models.SSyncRange) {
	log.Infof("Start sync host info ...")
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_START, "", task.UserCred)

	regions := driver.GetIRegions()
	localRegions, remoteRegions, result := models.CloudregionManager.SyncRegions(ctx, task.UserCred, provider.Provider, regions)
	msg := result.Result()
	log.Infof("SyncRegion result: %s", msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}

	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	for i := 0; i < len(localRegions); i += 1 {
		if len(syncRange.Region) > 0 && !utils.IsInStringArray(remoteRegions[i].GetId(), syncRange.Region) {
			continue
		}

		localZones, remoteZones := syncRegionZones(ctx, provider, task, &localRegions[i], remoteRegions[i])

		syncRegionVPCs(ctx, provider, task, &localRegions[i], remoteRegions[i])

		if localZones != nil && remoteZones != nil {
			for j := 0; j < len(localZones); j += 1 {

				if len(syncRange.Zone) > 0 && !utils.IsInStringArray(remoteZones[j].GetId(), syncRange.Zone) {
					continue
				}
				syncZoneStorages(ctx, provider, task, &localZones[j], remoteZones[j])
				syncZoneHosts(ctx, provider, task, &localZones[j], remoteZones[j], syncRange)
			}
		}
	}
}

func syncRegionZones(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion) ([]models.SZone, []cloudprovider.ICloudZone) {
	zones, err := remoteRegion.GetIZones()
	if err != nil {
		msg := fmt.Sprintf("GetZones for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return nil, nil
	}
	localZones, remoteZones, result := models.ZoneManager.SyncZones(ctx, task.UserCred, localRegion, zones)
	msg := result.Result()
	log.Infof("SyncZones for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return nil, nil
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	return localZones, remoteZones
}

func syncRegionVPCs(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion) {
	vpcs, err := remoteRegion.GetIVpcs()
	if err != nil {
		msg := fmt.Sprintf("GetVpcs for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}

	localVpcs, remoteVpcs, result := models.VpcManager.SyncVPCs(ctx, task.UserCred, localRegion, vpcs)
	msg := result.Result()
	log.Infof("SyncVPCs for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)

	for j := 0; j < len(localVpcs); j += 1 {
		syncVpcWires(ctx, provider, task, &localVpcs[j], remoteVpcs[j])
	}
}

func syncVpcWires(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localVpc *models.SVpc, remoteVpc cloudprovider.ICloudVpc) {
	wires, err := remoteVpc.GetIWires()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for vps %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localWires, remoteWires, result := models.WireManager.SyncWires(ctx, task.UserCred, localVpc, wires)
	msg := result.Result()
	log.Infof("SyncWires for VPC %s result: %s", localVpc.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	for i := 0; i < len(localWires); i += 1 {
		syncWireNetworks(ctx, provider, task, &localWires[i], remoteWires[i])
	}
}

func syncWireNetworks(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localWire *models.SWire, remoteWire cloudprovider.ICloudWire) {
	nets, err := remoteWire.GetINetworks()
	if err != nil {
		msg := fmt.Sprintf("GetINetworks for wire %s failed %s", remoteWire.GetId(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	_, _, result := models.NetworkManager.SyncNetworks(ctx, task.UserCred, localWire, nets)
	msg := result.Result()
	log.Infof("SyncNetworks for wire %s result: %s", localWire.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}

func syncZoneStorages(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localZone *models.SZone, remoteZone cloudprovider.ICloudZone) {
	storages, err := remoteZone.GetIStorages()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localStorages, remoteStorages, result := models.StorageManager.SyncStorages(ctx, task.UserCred, localZone, storages)
	msg := result.Result()
	log.Infof("SyncZones for region %s result: %s", localZone.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)

	for i := 0; i < len(localStorages); i += 1 {
		syncStorageCaches(ctx, provider, task, &localStorages[i], remoteStorages[i])
		syncStorageDisks(ctx, provider, task, &localStorages[i], remoteStorages[i])
	}
}

func syncStorageCaches(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localStorage *models.SStorage, remoteStorage cloudprovider.ICloudStorage) {
	remoteCache := remoteStorage.GetIStoragecache()
	localCache, err := models.StoragecacheManager.SyncWithCloudStoragecache(remoteCache)
	if err != nil {
		msg := fmt.Sprintf("SyncWithCloudStoragecache for storage %s failed %s", remoteStorage.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	err = localStorage.SetStoragecache(localCache)
	if err != nil {
		msg := fmt.Sprintf("localStorage %s set cache failed: %s", localStorage.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
	}
}

func syncStorageDisks(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localStorage *models.SStorage, remoteStorage cloudprovider.ICloudStorage) {
	disks, err := remoteStorage.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for storage %s failed %s", remoteStorage.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	_, _, result := models.DiskManager.SyncDisks(ctx, task.UserCred, localStorage, disks)
	msg := result.Result()
	log.Infof("SyncDisks for storage %s result: %s", localStorage.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}

func syncZoneHosts(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localZone *models.SZone, remoteZone cloudprovider.ICloudZone, syncRange *models.SSyncRange) {
	hosts, err := remoteZone.GetIHosts()
	if err != nil {
		msg := fmt.Sprintf("GetIHosts for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localHosts, remoteHosts, result := models.HostManager.SyncHosts(ctx, task.UserCred, localZone, hosts)
	msg := result.Result()
	log.Infof("SyncHosts for zone %s result: %s", localZone.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)

	for i := 0; i < len(localHosts); i += 1 {
		if len(syncRange.Host) > 0 && !utils.IsInStringArray(remoteHosts[i].GetGlobalId(), syncRange.Host) {
			continue
		}
		syncHostStorages(ctx, provider, task, &localHosts[i], remoteHosts[i])
		syncHostWires(ctx, provider, task, &localHosts[i], remoteHosts[i])
		syncHostVMs(ctx, provider, task, &localHosts[i], remoteHosts[i])
	}
}

func syncHostStorages(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localHost *models.SHost, remoteHost cloudprovider.ICloudHost) {
	storages, err := remoteHost.GetIStorages()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localHost.SyncHostStorages(ctx, task.UserCred, storages)
	msg := result.Result()
	log.Infof("SyncHostStorages for host %s result: %s", localHost.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}

func syncHostWires(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localHost *models.SHost, remoteHost cloudprovider.ICloudHost) {
	wires, err := remoteHost.GetIWires()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localHost.SyncHostWires(ctx, task.UserCred, wires)
	msg := result.Result()
	log.Infof("SyncHostWires for host %s result: %s", localHost.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}

func syncHostVMs(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localHost *models.SHost, remoteHost cloudprovider.ICloudHost) {
	vms, err := remoteHost.GetIVMs()
	if err != nil {
		msg := fmt.Sprintf("GetIVMs for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localVMs, remoteVMs, result := localHost.SyncHostVMs(ctx, task.UserCred, vms)
	msg := result.Result()
	log.Infof("SyncHostVMs for host %s result: %s", localHost.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)

	for i := 0; i < len(localVMs); i += 1 {
		syncVMNics(ctx, provider, task, localHost, &localVMs[i], remoteVMs[i])
		syncVMDisks(ctx, provider, task, localHost, &localVMs[i], remoteVMs[i])
	}
}

func syncVMNics(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, host *models.SHost, localVM *models.SGuest, remoteVM cloudprovider.ICloudVM) {
	nics, err := remoteVM.GetINics()
	if err != nil {
		msg := fmt.Sprintf("GetINics for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localVM.SyncVMNics(ctx, task.UserCred, host, nics)
	msg := result.Result()
	log.Infof("syncVMNics for VM %s result: %s", localVM.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}

func syncVMDisks(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, host *models.SHost, localVM *models.SGuest, remoteVM cloudprovider.ICloudVM) {
	disks, err := remoteVM.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localVM.SyncVMDisks(ctx, task.UserCred, host, disks)
	msg := result.Result()
	log.Infof("syncVMNics for VM %s result: %s", localVM.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}
