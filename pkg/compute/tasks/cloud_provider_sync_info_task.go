package tasks

import (
	"context"
	"fmt"

	"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/utils"
)

type CloudProviderSyncInfoTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudProviderSyncInfoTask{})
}

func getAction(params *jsonutils.JSONDict) string {
	fullSync := jsonutils.QueryBoolean(params, "full_sync", false)
	if !fullSync {
		syncRangeJson, _ := params.Get("sync_range")
		if syncRangeJson != nil {
			fullSync = jsonutils.QueryBoolean(syncRangeJson, "full_sync", false)
		}
	}

	action := ""

	if fullSync {
		action = logclient.ACT_CLOUD_FULLSYNC
	} else {
		action = logclient.ACT_CLOUD_SYNC
	}
	return action
}

func taskFail(ctx context.Context, task *CloudProviderSyncInfoTask, provider *models.SCloudprovider, reason string) {
	provider.SetStatus(task.UserCred, models.CLOUD_PROVIDER_DISCONNECTED, reason)
	task.SetStageFailed(ctx, reason)
}

func (self *CloudProviderSyncInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)
	provider.MarkStartSync(self.UserCred)
	// do sync

	notes := fmt.Sprintf("Start sync cloud provider %s status ...", provider.Name)
	log.Infof(notes)
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
		if err == nil && syncRange.NeedSyncInfo() {
			syncRange.Normalize()
			syncCloudProviderInfo(ctx, provider, self, driver, &syncRange)
		}
	}

	provider.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
	provider.CleanSchedCache()
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLog(provider, getAction(self.Params), body, self.UserCred, true)
}

func logSyncFailed(provider *models.SCloudprovider, task taskman.ITask, reason string) {
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, reason, task.GetUserCred())
	logclient.AddActionLog(provider, getAction(task.GetParams()), reason, task.GetUserCred(), false)
}

func syncCloudProviderInfo(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, syncRange *models.SSyncRange) {
	notes := fmt.Sprintf("Start sync host info ...")
	log.Infof(notes)
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_START, "", task.UserCred)

	if driver.IsOnPremiseInfrastructure() {
		syncOnPremiseCloudProviderInfo(ctx, provider, task, driver, syncRange)
	} else {
		syncPublicCloudProviderInfo(ctx, provider, task, driver, syncRange)
	}
}

func syncPublicCloudProviderInfo(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, syncRange *models.SSyncRange) {
	regions := driver.GetIRegions()

	// 华为云有点特殊一个provider只对应一个region
	providerPrefix := provider.Provider
	if providerPrefix == models.CLOUD_PROVIDER_HUAWEI {
		providerPrefix = providerPrefix + "/" + strings.Split(provider.Name, "_")[0]
	}

	localRegions, remoteRegions, result := models.CloudregionManager.SyncRegions(ctx, task.UserCred, providerPrefix, regions)
	msg := result.Result()
	log.Infof("SyncRegion result: %s", msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}

	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), "", task.UserCred, true)
	for i := 0; i < len(localRegions); i += 1 {
		if len(syncRange.Region) > 0 && !utils.IsInStringArray(localRegions[i].Id, syncRange.Region) {
			continue
		}
		syncRegionEips(ctx, provider, task, &localRegions[i], remoteRegions[i], syncRange)

		localZones, remoteZones := syncRegionZones(ctx, provider, task, &localRegions[i], remoteRegions[i])

		syncRegionVPCs(ctx, provider, task, &localRegions[i], remoteRegions[i], syncRange)

		if localZones != nil && remoteZones != nil {
			for j := 0; j < len(localZones); j += 1 {

				if len(syncRange.Zone) > 0 && !utils.IsInStringArray(localZones[j].Id, syncRange.Zone) {
					continue
				}
				syncZoneStorages(ctx, provider, task, &localZones[j], remoteZones[j], syncRange)
				syncZoneHosts(ctx, provider, task, &localZones[j], remoteZones[j], syncRange)
			}
		}
		syncRegionSnapshots(ctx, provider, task, &localRegions[i], remoteRegions[i], syncRange)
		syncRegionLoadbalancerAcls(ctx, provider, task, &localRegions[i], remoteRegions[i], syncRange)
		syncRegionLoadbalancerCertificates(ctx, provider, task, &localRegions[i], remoteRegions[i], syncRange)
		syncRegionLoadbalancers(ctx, provider, task, &localRegions[i], remoteRegions[i], syncRange)
	}
}

func syncRegionLoadbalancerCertificates(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) {
	certificates, err := remoteRegion.GetILoadbalancerCertificates()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerCertificates for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := models.LoadbalancerCertificateManager.SyncLoadbalancerCertificates(ctx, task.GetUserCred(), provider, localRegion, certificates, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerCertificates for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
}

func syncRegionLoadbalancerAcls(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) {
	acls, err := remoteRegion.GetILoadbalancerAcls()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerAcls for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := models.LoadbalancerAclManager.SyncLoadbalancerAcls(ctx, task.GetUserCred(), provider, localRegion, acls, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerAcls for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
}

func syncRegionLoadbalancers(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) {
	lbs, err := remoteRegion.GetILoadBalancers()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancers for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localLbs, remoteLbs, result := models.LoadbalancerManager.SyncLoadbalancers(ctx, task.GetUserCred(), provider, localRegion, lbs, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancers for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_LB_COMPLETE, msg, task.GetUserCred())
	for i := 0; i < len(localLbs); i++ {
		syncLoadbalancerBackendgroups(ctx, provider, task, &localLbs[i], remoteLbs[i], syncRange)
		syncLoadbalancerListeners(ctx, provider, task, &localLbs[i], remoteLbs[i], syncRange)
	}
}

func syncLoadbalancerListeners(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localLoadbalancer *models.SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *models.SSyncRange) {
	remoteListeners, err := remoteLoadbalancer.GetILoadbalancerListeners()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerListeners for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localListeners, remoteListeners, result := models.LoadbalancerListenerManager.SyncLoadbalancerListeners(ctx, task.GetUserCred(), provider, localLoadbalancer, remoteListeners, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerListeners for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	for i := 0; i < len(localListeners); i++ {
		syncLoadbalancerListenerRules(ctx, provider, task, &localListeners[i], remoteListeners[i], syncRange)
	}
}

func syncLoadbalancerListenerRules(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localListener *models.SLoadbalancerListener, remoteListener cloudprovider.ICloudLoadbalancerListener, syncRange *models.SSyncRange) {
	remoteRules, err := remoteListener.GetILoadbalancerListenerRules()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerListenerRules for listener %s failed %s", localListener.Name, err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := models.LoadbalancerListenerRuleManager.SyncLoadbalancerListenerRules(ctx, task.GetUserCred(), provider, localListener, remoteRules, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerListenerRules for listener %s result: %s", localListener.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
}

func syncLoadbalancerBackendgroups(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localLoadbalancer *models.SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *models.SSyncRange) {
	remoteBackendgroups, err := remoteLoadbalancer.GetILoadbalancerBackendGroups()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackendGroups for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localLbbgs, remoteLbbgs, result := models.LoadbalancerBackendGroupManager.SyncLoadbalancerBackendgroups(ctx, task.GetUserCred(), provider, localLoadbalancer, remoteBackendgroups, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerBackendgroups for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	for i := 0; i < len(localLbbgs); i++ {
		syncLoadbalancerBackends(ctx, provider, task, &localLbbgs[i], remoteLbbgs[i], syncRange)
	}
}

func syncLoadbalancerBackends(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localLbbg *models.SLoadbalancerBackendGroup, remoteLbbg cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *models.SSyncRange) {
	remoteLbbs, err := remoteLbbg.GetILoadbalancerBackends()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackends for lbbg %s failed %s", localLbbg.Name, err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := models.LoadbalancerBackendManager.SyncLoadbalancerBackends(ctx, task.GetUserCred(), provider, localLbbg, remoteLbbs, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerBackends for LoadbalancerBackendgroup %s result: %s", localLbbg.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
}

func syncRegionSnapshots(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) {
	snapshots, err := remoteRegion.GetISnapshots()
	if err != nil {
		msg := fmt.Sprintf("GetISnapshots for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}

	result := models.SnapshotManager.SyncSnapshots(ctx, task.GetUserCred(), provider, localRegion, snapshots, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	log.Infof("SyncSnapshots for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.GetUserCred())
}

func syncRegionEips(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) {
	eips, err := remoteRegion.GetIEips()
	if err != nil {
		msg := fmt.Sprintf("GetIEips for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}

	result := models.ElasticipManager.SyncEips(ctx, task.UserCred, provider, localRegion, eips, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	log.Infof("SyncEips for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
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
	notes := fmt.Sprintf("SyncZones for region %s result: %s", localRegion.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return nil, nil
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	return localZones, remoteZones
}

func syncRegionVPCs(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) {
	vpcs, err := remoteRegion.GetIVpcs()
	if err != nil {
		msg := fmt.Sprintf("GetVpcs for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}

	localVpcs, remoteVpcs, result := models.VpcManager.SyncVPCs(ctx, task.UserCred, provider, localRegion, vpcs)
	msg := result.Result()
	notes := fmt.Sprintf("SyncVPCs for region %s result: %s", localRegion.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	for j := 0; j < len(localVpcs); j += 1 {
		syncVpcWires(ctx, provider, task, &localVpcs[j], remoteVpcs[j], syncRange)
		syncVpcSecGroup(ctx, provider, task, &localVpcs[j], remoteVpcs[j], syncRange)
		syncVpcRouteTables(ctx, provider, task, &localVpcs[j], remoteVpcs[j], syncRange)
	}
}

func syncVpcSecGroup(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localVpc *models.SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *models.SSyncRange) {
	if secgroups, err := remoteVpc.GetISecurityGroups(); err != nil {
		msg := fmt.Sprintf("GetISecurityGroups for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	} else {
		_, _, result := models.SecurityGroupManager.SyncSecgroups(ctx, task.UserCred, secgroups, localVpc, provider.ProjectId, syncRange.ProjectSync)
		msg := result.Result()
		notes := fmt.Sprintf("SyncSecurityGroup for VPC %s result: %s", localVpc.Name, msg)
		log.Infof(notes)
		if result.IsError() {
			logSyncFailed(provider, task, msg)
			return
		}
	}
}

func syncVpcRouteTables(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localVpc *models.SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *models.SSyncRange) {
	routeTables, err := remoteVpc.GetIRouteTables()
	if err != nil {
		msg := fmt.Sprintf("GetIRouteTables for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	_, _, result := models.RouteTableManager.SyncRouteTables(ctx, task.GetUserCred(), localVpc, routeTables)
	msg := result.Result()
	notes := fmt.Sprintf("SyncRouteTables for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
}

func syncVpcWires(ctx context.Context, provider *models.SCloudprovider, task taskman.ITask, localVpc *models.SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *models.SSyncRange) {
	wires, err := remoteVpc.GetIWires()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localWires, remoteWires, result := models.WireManager.SyncWires(ctx, task.GetUserCred(), localVpc, wires)
	msg := result.Result()
	notes := fmt.Sprintf("SyncWires for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.GetUserCred())
	logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
	for i := 0; i < len(localWires); i += 1 {
		syncWireNetworks(ctx, provider, task, &localWires[i], remoteWires[i], syncRange)
	}
}

func syncWireNetworks(ctx context.Context, provider *models.SCloudprovider, task taskman.ITask, localWire *models.SWire, remoteWire cloudprovider.ICloudWire, syncRange *models.SSyncRange) {
	nets, err := remoteWire.GetINetworks()
	if err != nil {
		msg := fmt.Sprintf("GetINetworks for wire %s failed %s", remoteWire.GetId(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	_, _, result := models.NetworkManager.SyncNetworks(ctx, task.GetUserCred(), localWire, nets, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncNetworks for wire %s result: %s", localWire.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.GetUserCred())
	logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
}

func syncZoneStorages(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localZone *models.SZone, remoteZone cloudprovider.ICloudZone, syncRange *models.SSyncRange) {
	storages, err := remoteZone.GetIStorages()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localStorages, remoteStorages, result := models.StorageManager.SyncStorages(ctx, task.UserCred, provider, localZone, storages)
	msg := result.Result()
	notes := fmt.Sprintf("SyncZones for region %s result: %s", localZone.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)

	for i := 0; i < len(localStorages); i += 1 {
		syncStorageCaches(ctx, provider, task, &localStorages[i], remoteStorages[i])
		syncStorageDisks(ctx, provider, task, &localStorages[i], remoteStorages[i], syncRange)
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

func syncStorageDisks(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localStorage *models.SStorage, remoteStorage cloudprovider.ICloudStorage, syncRange *models.SSyncRange) {
	disks, err := remoteStorage.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for storage %s failed %s", remoteStorage.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	_, _, result := models.DiskManager.SyncDisks(ctx, task.UserCred, localStorage, disks, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncDisks for storage %s result: %s", localStorage.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncZoneHosts(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localZone *models.SZone, remoteZone cloudprovider.ICloudZone, syncRange *models.SSyncRange) {
	hosts, err := remoteZone.GetIHosts()
	if err != nil {
		msg := fmt.Sprintf("GetIHosts for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localHosts, remoteHosts, result := models.HostManager.SyncHosts(ctx, task.UserCred, provider, localZone, hosts, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHosts for zone %s result: %s", localZone.Name, msg)
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
		syncHostWires(ctx, provider, task, &localHosts[i], remoteHosts[i])
		syncHostVMs(ctx, provider, task, &localHosts[i], remoteHosts[i], syncRange)
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
	localStorages, remoteStorages, result := localHost.SyncHostStorages(ctx, task.UserCred, storages)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostStorages for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)

	for i := 0; i < len(localStorages); i += 1 {
		syncStorageCaches(ctx, provider, task, &localStorages[i], remoteStorages[i])
	}
}

func syncHostWires(ctx context.Context, provider *models.SCloudprovider, task taskman.ITask, localHost *models.SHost, remoteHost cloudprovider.ICloudHost) {
	wires, err := remoteHost.GetIWires()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localHost.SyncHostWires(ctx, task.GetUserCred(), wires)
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

func syncHostVMs(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localHost *models.SHost, remoteHost cloudprovider.ICloudHost, syncRange *models.SSyncRange) {
	vms, err := remoteHost.GetIVMs()
	if err != nil {
		msg := fmt.Sprintf("GetIVMs for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	localVMs, remoteVMs, result := localHost.SyncHostVMs(ctx, task.UserCred, vms, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostVMs for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	for i := 0; i < len(localVMs); i += 1 {
		syncVMNics(ctx, provider, task, localHost, &localVMs[i], remoteVMs[i])
		syncVMDisks(ctx, provider, task, localHost, &localVMs[i], remoteVMs[i], syncRange)
		syncVMEip(ctx, provider, task, &localVMs[i], remoteVMs[i])

		if localVMs[i].Status == models.VM_RUNNING {
			db.OpsLog.LogEvent(&localVMs[i], db.ACT_START, localVMs[i].GetShortDesc(ctx), task.UserCred)
		}
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
	notes := fmt.Sprintf("syncVMNics for VM %s result: %s", localVM.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncVMDisks(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, host *models.SHost, localVM *models.SGuest, remoteVM cloudprovider.ICloudVM, syncRange *models.SSyncRange) {
	disks, err := remoteVM.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localVM.SyncVMDisks(ctx, task.UserCred, host, disks, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("syncVMDisks for VM %s result: %s", localVM.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncVMEip(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localVM *models.SGuest, remoteVM cloudprovider.ICloudVM) {
	eip, err := remoteVM.GetIEIP()
	if err != nil {
		msg := fmt.Sprintf("GetIEIP for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localVM.SyncVMEip(ctx, task.UserCred, eip, provider.ProjectId)
	msg := result.Result()
	log.Infof("syncVMEip for VM %s result: %s", localVM.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}
