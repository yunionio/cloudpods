package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"
)

type SSyncableBaseResource struct {
	SyncStatus    string    `width:"10" charset:"ascii" default:"idle" list:"admin"`
	LastSync      time.Time `list:"admin"` // = Column(DateTime, nullable=True)
	LastSyncEndAt time.Time `list:"admin"`
}

func (self *SSyncableBaseResource) CanSync() bool {
	if self.SyncStatus == CLOUD_PROVIDER_SYNC_STATUS_QUEUED || self.SyncStatus == CLOUD_PROVIDER_SYNC_STATUS_SYNCING {
		if self.LastSync.IsZero() || time.Now().Sub(self.LastSync) > 900*time.Second {
			return true
		} else {
			return false
		}
	} else {
		return true
	}
}

type sStoragecacheSyncPair struct {
	local  *SStoragecache
	remote cloudprovider.ICloudStoragecache
}

func (pair *sStoragecacheSyncPair) syncCloudImages(ctx context.Context, userCred mcclient.TokenCredential) compare.SyncResult {
	return pair.local.SyncCloudImages(ctx, userCred, pair.remote)
}

func isInCache(pairs []sStoragecacheSyncPair, localCacheId string) bool {
	log.Debugf("isInCache %d %s", len(pairs), localCacheId)
	for i := range pairs {
		if pairs[i].local.Id == localCacheId {
			return true
		}
	}
	return false
}

func syncRegionZones(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) ([]SZone, []cloudprovider.ICloudZone, error) {
	zones, err := remoteRegion.GetIZones()
	if err != nil {
		msg := fmt.Sprintf("GetZones for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return nil, nil, err
	}
	localZones, remoteZones, result := ZoneManager.SyncZones(ctx, userCred, localRegion, zones)
	msg := result.Result()
	notes := fmt.Sprintf("SyncZones for region %s result: %s", localRegion.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return nil, nil, fmt.Errorf(msg)
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	return localZones, remoteZones, nil
}

/*func syncRegionSkus(localRegion *SCloudregion) {
	if localRegion == nil {
		log.Debugf("local region is nil skipped.")
		return
	}

	regionId := localRegion.GetId()
	if len(regionId) > 0 && ServerSkuManager.GetSkuCountByRegion(regionId) == 0 {
		// 提前同步instance type.如果同步失败可能导致vm 内存显示为0
		if err := skus.SyncSkusByRegion(localRegion); err != nil {
			msg := fmt.Sprintf("Get Skus for region %s failed %s", localRegion.GetName(), err)
			log.Errorf(msg)
			// 暂时不终止同步
			// logSyncFailed(provider, task, msg)
			return
		}
	}
}*/

func syncRegionEips(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	eips, err := remoteRegion.GetIEips()
	if err != nil {
		msg := fmt.Sprintf("GetIEips for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := ElasticipManager.SyncEips(ctx, userCred, provider, localRegion, eips, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	log.Infof("SyncEips for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncRegionVPCs(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	vpcs, err := remoteRegion.GetIVpcs()
	if err != nil {
		msg := fmt.Sprintf("GetVpcs for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	localVpcs, remoteVpcs, result := VpcManager.SyncVPCs(ctx, userCred, provider, localRegion, vpcs)
	msg := result.Result()
	notes := fmt.Sprintf("SyncVPCs for region %s result: %s", localRegion.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	for j := 0; j < len(localVpcs); j += 1 {
		func() {
			// lock vpc
			lockman.LockObject(ctx, &localVpcs[j])
			defer lockman.ReleaseObject(ctx, &localVpcs[j])

			syncVpcWires(ctx, userCred, provider, &localVpcs[j], remoteVpcs[j], syncRange)
			syncVpcSecGroup(ctx, userCred, provider, &localVpcs[j], remoteVpcs[j], syncRange)
			syncVpcRouteTables(ctx, userCred, provider, &localVpcs[j], remoteVpcs[j], syncRange)

		}()
	}
}

func syncVpcSecGroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	secgroups, err := remoteVpc.GetISecurityGroups()
	if err != nil {
		msg := fmt.Sprintf("GetISecurityGroups for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}

	_, _, result := SecurityGroupManager.SyncSecgroups(ctx, userCred, secgroups, localVpc, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncSecurityGroup for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}

}

func syncVpcRouteTables(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	routeTables, err := remoteVpc.GetIRouteTables()
	if err != nil {
		msg := fmt.Sprintf("GetIRouteTables for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}
	_, _, result := RouteTableManager.SyncRouteTables(ctx, userCred, localVpc, routeTables)
	msg := result.Result()
	notes := fmt.Sprintf("SyncRouteTables for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
}

func syncVpcWires(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	wires, err := remoteVpc.GetIWires()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}
	localWires, remoteWires, result := WireManager.SyncWires(ctx, userCred, localVpc, wires)
	msg := result.Result()
	notes := fmt.Sprintf("SyncWires for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
	for i := 0; i < len(localWires); i += 1 {
		func() {
			lockman.LockObject(ctx, &localWires[i])
			defer lockman.ReleaseObject(ctx, &localWires[i])

			syncWireNetworks(ctx, userCred, provider, &localWires[i], remoteWires[i], syncRange)
		}()
	}
}

func syncWireNetworks(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localWire *SWire, remoteWire cloudprovider.ICloudWire, syncRange *SSyncRange) {
	nets, err := remoteWire.GetINetworks()
	if err != nil {
		msg := fmt.Sprintf("GetINetworks for wire %s failed %s", remoteWire.GetId(), err)
		log.Errorf(msg)
		return
	}
	_, _, result := NetworkManager.SyncNetworks(ctx, userCred, localWire, nets, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncNetworks for wire %s result: %s", localWire.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
}

func syncZoneStorages(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localZone *SZone, remoteZone cloudprovider.ICloudZone, syncRange *SSyncRange, storageCachePairs []sStoragecacheSyncPair) []sStoragecacheSyncPair {
	storages, err := remoteZone.GetIStorages()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		return nil
	}
	localStorages, remoteStorages, result := StorageManager.SyncStorages(ctx, userCred, provider, localZone, storages)
	msg := result.Result()
	notes := fmt.Sprintf("SyncZones for region %s result: %s", localZone.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return nil
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)

	newCacheIds := make([]sStoragecacheSyncPair, 0)
	for i := 0; i < len(localStorages); i += 1 {
		func() {
			lockman.LockObject(ctx, &localStorages[i])
			defer lockman.ReleaseObject(ctx, &localStorages[i])

			if !isInCache(storageCachePairs, localStorages[i].StoragecacheId) && !isInCache(newCacheIds, localStorages[i].StoragecacheId) {
				cachePair := syncStorageCaches(ctx, userCred, provider, &localStorages[i], remoteStorages[i])
				if cachePair.remote != nil && cachePair.local != nil {
					newCacheIds = append(newCacheIds, cachePair)
				}
			}
			syncStorageDisks(ctx, userCred, provider, driver, &localStorages[i], remoteStorages[i], syncRange)

		}()
	}
	return newCacheIds
}

func syncStorageCaches(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localStorage *SStorage, remoteStorage cloudprovider.ICloudStorage) (cachePair sStoragecacheSyncPair) {
	remoteCache := remoteStorage.GetIStoragecache()
	localCache, err := StoragecacheManager.SyncWithCloudStoragecache(ctx, userCred, remoteCache)
	if err != nil {
		msg := fmt.Sprintf("SyncWithCloudStoragecache for storage %s failed %s", remoteStorage.GetName(), err)
		log.Errorf(msg)
		return
	}
	err = localStorage.SetStoragecache(userCred, localCache)
	if err != nil {
		msg := fmt.Sprintf("localStorage %s set cache failed: %s", localStorage.GetName(), err)
		log.Errorf(msg)
	}
	cachePair.local = localCache
	cachePair.remote = remoteCache
	return
}

func syncStorageDisks(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localStorage *SStorage, remoteStorage cloudprovider.ICloudStorage, syncRange *SSyncRange) {
	disks, err := remoteStorage.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for storage %s failed %s", remoteStorage.GetName(), err)
		log.Errorf(msg)
		return
	}
	_, _, result := DiskManager.SyncDisks(ctx, userCred, driver, localStorage, disks, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncDisks for storage %s result: %s", localStorage.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncZoneHosts(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localZone *SZone, remoteZone cloudprovider.ICloudZone, syncRange *SSyncRange, storageCachePairs []sStoragecacheSyncPair) []sStoragecacheSyncPair {
	hosts, err := remoteZone.GetIHosts()
	if err != nil {
		msg := fmt.Sprintf("GetIHosts for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		return nil
	}
	localHosts, remoteHosts, result := HostManager.SyncHosts(ctx, userCred, provider, localZone, hosts, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHosts for zone %s result: %s", localZone.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return nil
	}
	var newCachePairs []sStoragecacheSyncPair
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	for i := 0; i < len(localHosts); i += 1 {
		if len(syncRange.Host) > 0 && !utils.IsInStringArray(localHosts[i].Id, syncRange.Host) {
			continue
		}
		func() {
			lockman.LockObject(ctx, &localHosts[i])
			defer lockman.ReleaseObject(ctx, &localHosts[i])

			newCachePairs = syncHostStorages(ctx, userCred, provider, &localHosts[i], remoteHosts[i], storageCachePairs)
			syncHostWires(ctx, userCred, provider, &localHosts[i], remoteHosts[i])
			syncHostVMs(ctx, userCred, provider, driver, &localHosts[i], remoteHosts[i], syncRange)
		}()
	}
	return newCachePairs
}

func syncHostStorages(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localHost *SHost, remoteHost cloudprovider.ICloudHost, storageCachePairs []sStoragecacheSyncPair) []sStoragecacheSyncPair {
	storages, err := remoteHost.GetIStorages()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		return nil
	}
	localStorages, remoteStorages, result := localHost.SyncHostStorages(ctx, userCred, storages)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostStorages for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return nil
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)

	newCacheIds := make([]sStoragecacheSyncPair, 0)
	for i := 0; i < len(localStorages); i += 1 {
		if !isInCache(storageCachePairs, localStorages[i].StoragecacheId) && !isInCache(newCacheIds, localStorages[i].StoragecacheId) {
			cachePair := syncStorageCaches(ctx, userCred, provider, &localStorages[i], remoteStorages[i])
			if cachePair.remote != nil && cachePair.local != nil {
				newCacheIds = append(newCacheIds, cachePair)
			}
		}
	}
	return newCacheIds
}

func syncHostWires(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localHost *SHost, remoteHost cloudprovider.ICloudHost) {
	wires, err := remoteHost.GetIWires()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := localHost.SyncHostWires(ctx, userCred, wires)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostWires for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
}

func syncHostVMs(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localHost *SHost, remoteHost cloudprovider.ICloudHost, syncRange *SSyncRange) {
	vms, err := remoteHost.GetIVMs()
	if err != nil {
		msg := fmt.Sprintf("GetIVMs for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		return
	}
	localVMs, remoteVMs, result := localHost.SyncHostVMs(ctx, userCred, driver, vms, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostVMs for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	for i := 0; i < len(localVMs); i += 1 {
		func() {
			lockman.LockObject(ctx, &localVMs[i])
			defer lockman.ReleaseObject(ctx, &localVMs[i])

			syncVMNics(ctx, userCred, provider, localHost, &localVMs[i], remoteVMs[i])
			syncVMDisks(ctx, userCred, provider, driver, localHost, &localVMs[i], remoteVMs[i], syncRange)
			syncVMEip(ctx, userCred, provider, &localVMs[i], remoteVMs[i])

		}()

		if localVMs[i].Status == VM_RUNNING {
			db.OpsLog.LogEvent(&localVMs[i], db.ACT_START, localVMs[i].GetShortDesc(ctx), userCred)
		}
	}
}

func syncVMNics(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, host *SHost, localVM *SGuest, remoteVM cloudprovider.ICloudVM) {
	nics, err := remoteVM.GetINics()
	if err != nil {
		msg := fmt.Sprintf("GetINics for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := localVM.SyncVMNics(ctx, userCred, host, nics)
	msg := result.Result()
	notes := fmt.Sprintf("syncVMNics for VM %s result: %s", localVM.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncVMDisks(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, driver cloudprovider.ICloudProvider, host *SHost, localVM *SGuest, remoteVM cloudprovider.ICloudVM, syncRange *SSyncRange) {
	disks, err := remoteVM.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := localVM.SyncVMDisks(ctx, userCred, driver, host, disks, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("syncVMDisks for VM %s result: %s", localVM.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncVMEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localVM *SGuest, remoteVM cloudprovider.ICloudVM) {
	eip, err := remoteVM.GetIEIP()
	if err != nil {
		msg := fmt.Sprintf("GetIEIP for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := localVM.SyncVMEip(ctx, userCred, eip, provider.ProjectId)
	msg := result.Result()
	log.Infof("syncVMEip for VM %s result: %s", localVM.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncRegionSkusFromCloud(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localZone *SZone, remoteRegion cloudprovider.ICloudRegion, remoteZone cloudprovider.ICloudZone) {
	skus, err := remoteRegion.GetSkus(remoteZone.GetId())
	if err != nil {
		msg := fmt.Sprintf("GetSkus for zone %s failed %v", localZone.Name, err)
		log.Errorf(msg)
		return
	}

	result := ServerSkuManager.SyncCloudSkusByRegion(ctx, userCred, provider, localZone, skus)
	msg := result.Result()
	log.Infof("SyncCloudSkusByRegion for zone %s result: %s", localZone.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionLoadbalancerCertificates(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	certificates, err := remoteRegion.GetILoadBalancerCertificates()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerCertificates for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := LoadbalancerCertificateManager.SyncLoadbalancerCertificates(ctx, userCred, provider, localRegion, certificates, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerCertificates for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionLoadbalancerAcls(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	acls, err := remoteRegion.GetILoadBalancerAcls()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerAcls for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := LoadbalancerAclManager.SyncLoadbalancerAcls(ctx, userCred, provider, localRegion, acls, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerAcls for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionLoadbalancers(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	lbs, err := remoteRegion.GetILoadBalancers()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancers for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}
	localLbs, remoteLbs, result := LoadbalancerManager.SyncLoadbalancers(ctx, userCred, provider, localRegion, lbs, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancers for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_LB_COMPLETE, msg, userCred)
	for i := 0; i < len(localLbs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbs[i])
			defer lockman.ReleaseObject(ctx, &localLbs[i])

			syncLoadbalancerBackendgroups(ctx, userCred, provider, &localLbs[i], remoteLbs[i], syncRange)
			syncLoadbalancerListeners(ctx, userCred, provider, &localLbs[i], remoteLbs[i], syncRange)

		}()
	}
}

func syncLoadbalancerListeners(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) {
	remoteListeners, err := remoteLoadbalancer.GetILoadBalancerListeners()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerListeners for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorf(msg)
		return
	}
	localListeners, remoteListeners, result := LoadbalancerListenerManager.SyncLoadbalancerListeners(ctx, userCred, provider, localLoadbalancer, remoteListeners, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerListeners for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localListeners); i++ {
		func() {
			lockman.LockObject(ctx, &localListeners[i])
			defer lockman.ReleaseObject(ctx, &localListeners[i])

			syncLoadbalancerListenerRules(ctx, userCred, provider, &localListeners[i], remoteListeners[i], syncRange)

		}()
	}
}

func syncLoadbalancerListenerRules(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localListener *SLoadbalancerListener, remoteListener cloudprovider.ICloudLoadbalancerListener, syncRange *SSyncRange) {
	remoteRules, err := remoteListener.GetILoadbalancerListenerRules()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerListenerRules for listener %s failed %s", localListener.Name, err)
		log.Errorf(msg)
		return
	}
	result := LoadbalancerListenerRuleManager.SyncLoadbalancerListenerRules(ctx, userCred, provider, localListener, remoteRules, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerListenerRules for listener %s result: %s", localListener.Name, msg)
	if result.IsError() {
		return
	}
}

func syncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) {
	remoteBackendgroups, err := remoteLoadbalancer.GetILoadBalancerBackendGroups()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerBackendGroups for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorf(msg)
		return
	}
	localLbbgs, remoteLbbgs, result := LoadbalancerBackendGroupManager.SyncLoadbalancerBackendgroups(ctx, userCred, provider, localLoadbalancer, remoteBackendgroups, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerBackendgroups for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localLbbgs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbbgs[i])
			defer lockman.ReleaseObject(ctx, &localLbbgs[i])

			syncLoadbalancerBackends(ctx, userCred, provider, &localLbbgs[i], remoteLbbgs[i], syncRange)
		}()
	}
}

func syncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localLbbg *SLoadbalancerBackendGroup, remoteLbbg cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) {
	remoteLbbs, err := remoteLbbg.GetILoadbalancerBackends()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackends for lbbg %s failed %s", localLbbg.Name, err)
		log.Errorf(msg)
		return
	}
	result := LoadbalancerBackendManager.SyncLoadbalancerBackends(ctx, userCred, provider, localLbbg, remoteLbbs, syncRange)
	msg := result.Result()
	log.Infof("SyncLoadbalancerBackends for LoadbalancerBackendgroup %s result: %s", localLbbg.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionSnapshots(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	snapshots, err := remoteRegion.GetISnapshots()
	if err != nil {
		msg := fmt.Sprintf("GetISnapshots for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := SnapshotManager.SyncSnapshots(ctx, userCred, provider, localRegion, snapshots, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	log.Infof("SyncSnapshots for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncPublicCloudProviderInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	driver cloudprovider.ICloudProvider,
	localRegion *SCloudregion,
	remoteRegion cloudprovider.ICloudRegion,
	syncRange *SSyncRange,
) error {
	if syncRange != nil && len(syncRange.Region) > 0 && !utils.IsInStringArray(localRegion.Id, syncRange.Region) {
		// no need to sync
		return nil
	}

	log.Debugf("Start sync cloud provider %s(%s) on region %s(%s)",
		provider.Name, provider.Provider, remoteRegion.GetName(), remoteRegion.GetId())

	storageCachePairs := make([]sStoragecacheSyncPair, 0)

	localZones, remoteZones, _ := syncRegionZones(ctx, userCred, provider, localRegion, remoteRegion)

	if !driver.GetFactory().NeedSyncSkuFromCloud() {
		// syncRegionSkus(localRegion)
	}

	// no need to lock public cloud region as cloud region for public cloud is readonly

	syncRegionEips(ctx, userCred, provider, localRegion, remoteRegion, syncRange)

	syncRegionVPCs(ctx, userCred, provider, localRegion, remoteRegion, syncRange)

	for j := 0; j < len(localZones); j += 1 {

		if len(syncRange.Zone) > 0 && !utils.IsInStringArray(localZones[j].Id, syncRange.Zone) {
			continue
		}
		// no need to lock zone as public cloud zone is read-only

		newPairs := syncZoneStorages(ctx, userCred, provider, driver, &localZones[j], remoteZones[j], syncRange, storageCachePairs)
		if len(newPairs) > 0 {
			storageCachePairs = append(storageCachePairs, newPairs...)
		}
		newPairs = syncZoneHosts(ctx, userCred, provider, driver, &localZones[j], remoteZones[j], syncRange, storageCachePairs)
		if len(newPairs) > 0 {
			storageCachePairs = append(storageCachePairs, newPairs...)
		}

		if driver.GetFactory().NeedSyncSkuFromCloud() {
			syncRegionSkusFromCloud(ctx, userCred, provider, &localZones[j], remoteRegion, remoteZones[j])
		}
	}

	syncRegionSnapshots(ctx, userCred, provider, localRegion, remoteRegion, syncRange)

	syncRegionLoadbalancerAcls(ctx, userCred, provider, localRegion, remoteRegion, syncRange)
	syncRegionLoadbalancerCertificates(ctx, userCred, provider, localRegion, remoteRegion, syncRange)
	syncRegionLoadbalancers(ctx, userCred, provider, localRegion, remoteRegion, syncRange)

	log.Debugf("storageCachePairs count %d", len(storageCachePairs))
	for i := range storageCachePairs {
		result := storageCachePairs[i].syncCloudImages(ctx, userCred)
		msg := result.Result()
		log.Infof("syncCloudImages result: %s", msg)
	}

	return nil
}

func syncOnPremiseCloudProviderInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	driver cloudprovider.ICloudProvider,
	syncRange *SSyncRange,
) error {
	log.Debugf("Start sync on-premise provider %s(%s)", provider.Name, provider.Provider)
	iregion, err := driver.GetOnPremiseIRegion()
	if err != nil {
		msg := fmt.Sprintf("GetOnPremiseIRegion for provider %s failed %s", provider.GetName(), err)
		log.Errorf(msg)
		return err
	}

	ihosts, err := iregion.GetIHosts()
	if err != nil {
		msg := fmt.Sprintf("GetIHosts for provider %s failed %s", provider.GetName(), err)
		log.Errorf(msg)
		return err
	}

	localHosts, remoteHosts, result := HostManager.SyncHosts(ctx, userCred, provider, nil, ihosts, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHosts for provider %s result: %s", provider.Name, msg)
	log.Infof(notes)
	//if result.IsError() {
	//	logSyncFailed(provider, task, msg)
	// return
	//}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)

	storageCachePairs := make([]sStoragecacheSyncPair, 0)

	for i := 0; i < len(localHosts); i += 1 {
		if len(syncRange.Host) > 0 && !utils.IsInStringArray(localHosts[i].Id, syncRange.Host) {
			continue
		}
		newCachePairs := syncHostStorages(ctx, userCred, provider, &localHosts[i], remoteHosts[i], storageCachePairs)
		if len(newCachePairs) > 0 {
			storageCachePairs = append(storageCachePairs, newCachePairs...)
		}
		syncHostNics(ctx, userCred, provider, &localHosts[i], remoteHosts[i])
		syncHostVMs(ctx, userCred, provider, driver, &localHosts[i], remoteHosts[i], syncRange)
	}

	log.Debugf("storageCachePairs count %d", len(storageCachePairs))
	for i := range storageCachePairs {
		result := storageCachePairs[i].syncCloudImages(ctx, userCred)
		msg := result.Result()
		log.Infof("syncCloudImages result: %s", msg)
	}
	return nil
}

func syncHostNics(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localHost *SHost, remoteHost cloudprovider.ICloudHost) {
	result := localHost.SyncHostExternalNics(ctx, userCred, remoteHost)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostWires for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
}

func (manager *SCloudproviderregionManager) fetchRecordsByQuery(q *sqlchemy.SQuery) []SCloudproviderregion {
	recs := make([]SCloudproviderregion, 0)
	err := db.FetchModelObjects(manager, q, &recs)
	if err != nil {
		return nil
	}
	return recs
}

func (manager *SCloudproviderregionManager) initAllRecords() {
	recs := manager.fetchRecordsByQuery(manager.Query())
	for i := range recs {
		db.Update(&recs[i], func() error {
			recs[i].SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
	}
}

func (self *SCloudproviderregion) needSync() bool {
	if self.LastSyncEndAt.IsZero() {
		return true
	}
	if time.Now().Sub(self.LastSyncEndAt) > time.Duration(self.getSyncIntervalSeconds(nil))*time.Second {
		return true
	}
	return false
}
