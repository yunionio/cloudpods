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

package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SSyncableBaseResource struct {
	SyncStatus    string    `width:"10" charset:"ascii" default:"idle" list:"domain"`
	LastSync      time.Time `list:"domain"` // = Column(DateTime, nullable=True)
	LastSyncEndAt time.Time `list:"domain"`
}

type SSyncableBaseResourceManager struct{}

func (self *SSyncableBaseResource) CanSync() bool {
	if self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED || self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING {
		if self.LastSync.IsZero() || time.Now().Sub(self.LastSync) > time.Minute*30 {
			return true
		}
		return false
	}
	return true
}

func (manager *SSyncableBaseResourceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SyncableBaseResourceListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.SyncStatus) > 0 {
		q = q.In("sync_status", query.SyncStatus)
	}
	return q, nil
}

type sStoragecacheSyncPair struct {
	local  *SStoragecache
	region *SCloudregion
	remote cloudprovider.ICloudStoragecache
	isNew  bool
}

func (pair *sStoragecacheSyncPair) syncCloudImages(ctx context.Context, userCred mcclient.TokenCredential) compare.SyncResult {
	return pair.local.SyncCloudImages(ctx, userCred, pair.remote, pair.region)
}

func isInCache(pairs []sStoragecacheSyncPair, localCacheId string) bool {
	// log.Debugf("isInCache %d %s", len(pairs), localCacheId)
	for i := range pairs {
		if pairs[i].local.Id == localCacheId {
			return true
		}
	}
	return false
}

func syncRegionQuotas(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, driver cloudprovider.ICloudProvider, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	quotas, err := func() ([]cloudprovider.ICloudQuota, error) {
		defer syncResults.AddRequestCost(CloudproviderQuotaManager)()
		return remoteRegion.GetICloudQuotas()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudQuotas for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(CloudproviderQuotaManager)()
		return CloudproviderQuotaManager.SyncQuotas(ctx, userCred, provider.GetOwnerId(), provider, localRegion, api.CLOUD_PROVIDER_QUOTA_RANGE_CLOUDREGION, quotas)
	}()
	syncResults.Add(CloudproviderQuotaManager, result)
	msg := result.Result()
	notes := fmt.Sprintf("SyncQuotas for region %s result: %s", localRegion.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return fmt.Errorf(msg)
	}
	return nil
}

func syncRegionZones(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) ([]SZone, []cloudprovider.ICloudZone, error) {
	zones, err := func() ([]cloudprovider.ICloudZone, error) {
		defer syncResults.AddRequestCost(ZoneManager)()
		return remoteRegion.GetIZones()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetZones for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return nil, nil, err
	}
	localZones, remoteZones, result := func() ([]SZone, []cloudprovider.ICloudZone, compare.SyncResult) {
		defer syncResults.AddSqlCost(ZoneManager)()
		return ZoneManager.SyncZones(ctx, userCred, localRegion, zones)
	}()
	syncResults.Add(ZoneManager, result)
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

func syncRegionSkus(ctx context.Context, userCred mcclient.TokenCredential, localRegion *SCloudregion) {
	if localRegion == nil {
		log.Debugf("local region is nil, skipp...")
		return
	}

	regionId := localRegion.GetId()
	if len(regionId) == 0 {
		log.Debugf("local region Id is empty, skip...")
		return
	}

	cnt, err := ServerSkuManager.GetSkuCountByRegion(regionId)
	if err != nil {
		log.Errorf("GetSkuCountByRegion fail %s", err)
		return
	}

	if cnt == 0 {
		// 提前同步instance type.如果同步失败可能导致vm 内存显示为0
		if ret := SyncServerSkusByRegion(ctx, userCred, localRegion, nil); ret.IsError() {
			msg := fmt.Sprintf("Get Skus for region %s failed %s", localRegion.GetName(), ret.Result())
			log.Errorln(msg)
			// 暂时不终止同步
			// logSyncFailed(provider, task, msg)
			return
		}
	}

	if localRegion.GetDriver().IsSupportedElasticcache() {
		cnt, err = ElasticcacheSkuManager.GetSkuCountByRegion(regionId)
		if err != nil {
			log.Errorf("ElasticcacheSkuManager.GetSkuCountByRegion fail %s", err)
			return
		}

		if cnt == 0 {
			SyncElasticCacheSkusByRegion(ctx, userCred, localRegion)
		}
	}
}

func syncRegionEips(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	eips, err := func() ([]cloudprovider.ICloudEIP, error) {
		defer syncResults.AddRequestCost(ElasticipManager)()
		return remoteRegion.GetIEips()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIEips for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(ElasticipManager)()
		return ElasticipManager.SyncEips(ctx, userCred, provider, localRegion, eips, provider.GetOwnerId())
	}()

	syncResults.Add(ElasticipManager, result)

	msg := result.Result()
	log.Infof("SyncEips for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncRegionBuckets(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) {
	buckets, err := func() ([]cloudprovider.ICloudBucket, error) {
		defer syncResults.AddRequestCost(BucketManager)()
		return remoteRegion.GetIBuckets()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIBuckets for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(BucketManager)()
		return BucketManager.syncBuckets(ctx, userCred, provider, localRegion, buckets)
	}()

	syncResults.Add(BucketManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("GetIBuckets for region %s result: %s", localRegion.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncRegionVPCs(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	vpcs, err := func() ([]cloudprovider.ICloudVpc, error) {
		defer syncResults.AddRequestCost(VpcManager)()
		return remoteRegion.GetIVpcs()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetVpcs for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	localVpcs, remoteVpcs, result := func() ([]SVpc, []cloudprovider.ICloudVpc, compare.SyncResult) {
		defer syncResults.AddSqlCost(VpcManager)()
		return VpcManager.SyncVPCs(ctx, userCred, provider, localRegion, vpcs)
	}()

	syncResults.Add(VpcManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncVPCs for region %s result: %s", localRegion.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	globalVpcIds := []string{}
	for j := 0; j < len(localVpcs); j += 1 {
		func() {
			// lock vpc
			lockman.LockObject(ctx, &localVpcs[j])
			defer lockman.ReleaseObject(ctx, &localVpcs[j])

			if localVpcs[j].Deleted {
				return
			}

			syncVpcWires(ctx, userCred, syncResults, provider, &localVpcs[j], remoteVpcs[j], syncRange)
			if localRegion.GetDriver().IsSecurityGroupBelongVpc() ||
				(localRegion.GetDriver().IsSecurityGroupBelongGlobalVpc() && !utils.IsInStringArray(localVpcs[j].GlobalvpcId, globalVpcIds)) ||
				localRegion.GetDriver().IsSupportClassicSecurityGroup() || j == 0 { //有vpc属性的每次都同步,支持classic的vpc也同步，否则仅同步一次
				if len(localVpcs[j].GlobalvpcId) > 0 {
					globalVpcIds = append(globalVpcIds, localVpcs[j].GlobalvpcId)
				}
				syncVpcSecGroup(ctx, userCred, syncResults, provider, &localVpcs[j], remoteVpcs[j], syncRange)
			}
			syncVpcNatgateways(ctx, userCred, syncResults, provider, &localVpcs[j], remoteVpcs[j], syncRange)
			syncVpcPeerConnections(ctx, userCred, syncResults, provider, &localVpcs[j], remoteVpcs[j], syncRange)
			syncVpcRouteTables(ctx, userCred, syncResults, provider, &localVpcs[j], remoteVpcs[j], syncRange)
			syncIPv6Gateways(ctx, userCred, syncResults, provider, &localVpcs[j], remoteVpcs[j], syncRange)
		}()
	}
}

func syncRegionAccessGroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	accessGroups, err := func() ([]cloudprovider.ICloudAccessGroup, error) {
		defer syncResults.AddRequestCost(AccessGroupManager)()
		return remoteRegion.GetICloudAccessGroups()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
			return
		}
		log.Errorf("GetICloudFileSystems for region %s error: %v", localRegion.Name, err)
		return
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(AccessGroupManager)()
		return localRegion.SyncAccessGroups(ctx, userCred, provider, accessGroups)
	}()
	syncResults.Add(AccessGroupCacheManager, result)
	log.Infof("Sync Access Group Caches for region %s result: %s", localRegion.Name, result.Result())
}

func syncRegionFileSystems(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	filesystems, err := func() ([]cloudprovider.ICloudFileSystem, error) {
		defer syncResults.AddRequestCost(FileSystemManager)()
		return remoteRegion.GetICloudFileSystems()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
			return
		}
		log.Errorf("GetICloudFileSystems for region %s error: %v", localRegion.Name, err)
		return
	}

	localFSs, removeFSs, result := func() ([]SFileSystem, []cloudprovider.ICloudFileSystem, compare.SyncResult) {
		defer syncResults.AddSqlCost(FileSystemManager)()
		return localRegion.SyncFileSystems(ctx, userCred, provider, filesystems)
	}()
	syncResults.Add(FileSystemManager, result)
	log.Infof("Sync FileSystem for region %s result: %s", localRegion.Name, result.Result())

	for j := 0; j < len(localFSs); j += 1 {
		func() {
			// lock file system
			lockman.LockObject(ctx, &localFSs[j])
			defer lockman.ReleaseObject(ctx, &localFSs[j])

			if localFSs[j].Deleted {
				return
			}

			syncFileSystemMountTargets(ctx, userCred, &localFSs[j], removeFSs[j])
		}()
	}
}

func syncFileSystemMountTargets(ctx context.Context, userCred mcclient.TokenCredential, localFs *SFileSystem, remoteFs cloudprovider.ICloudFileSystem) {
	mountTargets, err := remoteFs.GetMountTargets()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
			return
		}
		log.Errorf("GetMountTargets for %s error: %v", localFs.Name, err)
		return
	}
	result := localFs.SyncMountTargets(ctx, userCred, mountTargets)
	log.Infof("SyncMountTargets for FileSystem %s result: %s", localFs.Name, result.Result())
}

func syncVpcPeerConnections(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	peerConnections, err := func() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
		defer syncResults.AddRequestCost(VpcPeeringConnectionManager)()
		return remoteVpc.GetICloudVpcPeeringConnections()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
			return
		}
		log.Errorf("GetICloudVpcPeeringConnections for vpc %s failed %v", localVpc.Name, err)
		return
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(VpcPeeringConnectionManager)()
		return localVpc.SyncVpcPeeringConnections(ctx, userCred, peerConnections)
	}()
	syncResults.Add(VpcPeeringConnectionManager, result)

	accepterPeerings, err := func() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
		defer syncResults.AddRequestCost(VpcPeeringConnectionManager)()
		return remoteVpc.GetICloudAccepterVpcPeeringConnections()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
			return
		}
		log.Errorf("GetICloudVpcPeeringConnections for vpc %s failed %v", localVpc.Name, err)
		return
	}
	backSyncResult := func() compare.SyncResult {
		defer syncResults.AddSqlCost(VpcPeeringConnectionManager)()
		return localVpc.BackSycVpcPeeringConnectionsVpc(accepterPeerings)
	}()
	syncResults.Add(VpcPeeringConnectionManager, backSyncResult)

	log.Infof("SyncVpcPeeringConnections for vpc %s result: %s", localVpc.Name, result.Result())
	if result.IsError() {
		return
	}
}

func syncVpcSecGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	secgroups, err := func() ([]cloudprovider.ICloudSecurityGroup, error) {
		defer syncResults.AddRequestCost(SecurityGroupManager)()
		return remoteVpc.GetISecurityGroups()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetISecurityGroups for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}

	_, _, result := func() ([]SSecurityGroup, []cloudprovider.ICloudSecurityGroup, compare.SyncResult) {
		defer syncResults.AddSqlCost(SecurityGroupManager)()
		return SecurityGroupCacheManager.SyncSecurityGroupCaches(ctx, userCred, provider, secgroups, localVpc)
	}()
	syncResults.Add(SecurityGroupCacheManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncSecurityGroupCaches for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
}

func syncVpcRouteTables(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	routeTables, err := func() ([]cloudprovider.ICloudRouteTable, error) {
		defer syncResults.AddRequestCost(RouteTableManager)()
		return remoteVpc.GetIRouteTables()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIRouteTables for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}
	localRouteTables, remoteRouteTables, result := func() ([]SRouteTable, []cloudprovider.ICloudRouteTable, compare.SyncResult) {
		defer syncResults.AddSqlCost(RouteTableManager)()
		return RouteTableManager.SyncRouteTables(ctx, userCred, localVpc, routeTables, provider)
	}()

	syncResults.Add(RouteTableManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncRouteTables for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localRouteTables); i++ {
		func() {
			lockman.LockObject(ctx, &localRouteTables[i])
			defer lockman.ReleaseObject(ctx, &localRouteTables[i])

			if localRouteTables[i].Deleted {
				return
			}
			localRouteTables[i].SyncRouteTableRouteSets(ctx, userCred, remoteRouteTables[i], provider)
			localRouteTables[i].SyncRouteTableAssociations(ctx, userCred, remoteRouteTables[i], provider)
		}()
	}
}

func syncIPv6Gateways(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	exts, err := func() ([]cloudprovider.ICloudIPv6Gateway, error) {
		defer syncResults.AddRequestCost(IPv6GatewayManager)()
		return remoteVpc.GetICloudIPv6Gateways()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudIPv6Gateways for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(IPv6GatewayManager)()
		return localVpc.SyncIPv6Gateways(ctx, userCred, exts, provider)
	}()

	syncResults.Add(IPv6GatewayManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncIPv6Gateways for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
}

func syncVpcNatgateways(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	natGateways, err := func() ([]cloudprovider.ICloudNatGateway, error) {
		defer syncResults.AddRequestCost(NatGatewayManager)()
		return remoteVpc.GetINatGateways()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented {
			return
		}
		msg := fmt.Sprintf("GetINatGateways for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}
	localNatGateways, remoteNatGateways, result := func() ([]SNatGateway, []cloudprovider.ICloudNatGateway, compare.SyncResult) {
		defer syncResults.AddSqlCost(NatGatewayManager)()
		return NatGatewayManager.SyncNatGateways(ctx, userCred, provider.GetOwnerId(), provider, localVpc, natGateways)
	}()

	syncResults.Add(NatGatewayManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncNatGateways for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}

	for i := 0; i < len(localNatGateways); i++ {
		func() {
			lockman.LockObject(ctx, &localNatGateways[i])
			defer lockman.ReleaseObject(ctx, &localNatGateways[i])

			if localNatGateways[i].Deleted {
				return
			}

			syncNatGatewayEips(ctx, userCred, provider, &localNatGateways[i], remoteNatGateways[i])
			syncNatDTable(ctx, userCred, provider, &localNatGateways[i], remoteNatGateways[i])
			syncNatSTable(ctx, userCred, provider, &localNatGateways[i], remoteNatGateways[i])
		}()
	}
}

func syncNatGatewayEips(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localNatGateway *SNatGateway, remoteNatGateway cloudprovider.ICloudNatGateway) {
	eips, err := remoteNatGateway.GetIEips()
	if err != nil {
		msg := fmt.Sprintf("GetIEIPs for NatGateway %s failed %s", remoteNatGateway.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := localNatGateway.SyncNatGatewayEips(ctx, userCred, provider, eips)
	msg := result.Result()
	log.Infof("SyncNatGatewayEips for NatGateway %s result: %s", localNatGateway.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncNatDTable(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localNatGateway *SNatGateway, remoteNatGateway cloudprovider.ICloudNatGateway) {
	dtable, err := remoteNatGateway.GetINatDTable()
	if err != nil {
		msg := fmt.Sprintf("GetINatDTable for NatGateway %s failed %s", remoteNatGateway.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := NatDEntryManager.SyncNatDTable(ctx, userCred, provider, localNatGateway, dtable)
	msg := result.Result()
	log.Infof("SyncNatDTable for NatGateway %s result: %s", localNatGateway.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncNatSTable(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localNatGateway *SNatGateway, remoteNatGateway cloudprovider.ICloudNatGateway) {
	stable, err := remoteNatGateway.GetINatSTable()
	if err != nil {
		msg := fmt.Sprintf("GetINatSTable for NatGateway %s failed %s", remoteNatGateway.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := NatSEntryManager.SyncNatSTable(ctx, userCred, provider, localNatGateway, stable)
	msg := result.Result()
	log.Infof("SyncNatSTable for NatGateway %s result: %s", localNatGateway.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncVpcWires(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localVpc *SVpc, remoteVpc cloudprovider.ICloudVpc, syncRange *SSyncRange) {
	wires, err := func() ([]cloudprovider.ICloudWire, error) {
		defer func() {
			if syncResults != nil {
				syncResults.AddRequestCost(WireManager)()
			}
		}()
		return remoteVpc.GetIWires()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for vpc %s failed %s", remoteVpc.GetId(), err)
		log.Errorf(msg)
		return
	}
	localWires, remoteWires, result := func() ([]SWire, []cloudprovider.ICloudWire, compare.SyncResult) {
		defer func() {
			if syncResults != nil {
				syncResults.AddSqlCost(WireManager)()
			}
		}()
		return WireManager.SyncWires(ctx, userCred, localVpc, wires, provider)
	}()

	if syncResults != nil {
		syncResults.Add(WireManager, result)
	}

	msg := result.Result()
	notes := fmt.Sprintf("SyncWires for VPC %s result: %s", localVpc.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
	for i := 0; i < len(localWires); i += 1 {
		func() {
			lockman.LockObject(ctx, &localWires[i])
			defer lockman.ReleaseObject(ctx, &localWires[i])

			if localWires[i].Deleted {
				return
			}
			syncWireNetworks(ctx, userCred, syncResults, provider, &localWires[i], remoteWires[i], syncRange)
		}()
	}
}

func syncWireNetworks(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localWire *SWire, remoteWire cloudprovider.ICloudWire, syncRange *SSyncRange) {
	nets, err := func() ([]cloudprovider.ICloudNetwork, error) {
		defer func() {
			if syncResults != nil {
				syncResults.AddRequestCost(NetworkManager)()
			}
		}()
		return remoteWire.GetINetworks()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetINetworks for wire %s failed %s", remoteWire.GetId(), err)
		log.Errorf(msg)
		return
	}
	_, _, result := func() ([]SNetwork, []cloudprovider.ICloudNetwork, compare.SyncResult) {
		defer func() {
			if syncResults != nil {
				syncResults.AddSqlCost(NetworkManager)()
			}
		}()
		return NetworkManager.SyncNetworks(ctx, userCred, localWire, nets, provider)
	}()

	if syncResults != nil {
		syncResults.Add(NetworkManager, result)
	}

	msg := result.Result()
	notes := fmt.Sprintf("SyncNetworks for wire %s result: %s", localWire.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
}

func syncZoneStorages(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localZone *SZone, remoteZone cloudprovider.ICloudZone, syncRange *SSyncRange, storageCachePairs []sStoragecacheSyncPair) []sStoragecacheSyncPair {
	storages, err := func() ([]cloudprovider.ICloudStorage, error) {
		defer syncResults.AddRequestCost(StorageManager)()
		return remoteZone.GetIStorages()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		return nil
	}
	localStorages, remoteStorages, result := func() ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
		defer syncResults.AddSqlCost(StorageManager)()
		return StorageManager.SyncStorages(ctx, userCred, provider, localZone, storages)
	}()

	syncResults.Add(StorageManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncStorages for zone %s result: %s", localZone.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return nil
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)

	newCacheIds := make([]sStoragecacheSyncPair, 0)
	for i := 0; i < len(localStorages); i += 1 {
		func() {
			lockman.LockObject(ctx, &localStorages[i])
			defer lockman.ReleaseObject(ctx, &localStorages[i])

			if localStorages[i].Deleted {
				return
			}

			if !isInCache(storageCachePairs, localStorages[i].StoragecacheId) && !isInCache(newCacheIds, localStorages[i].StoragecacheId) {
				cachePair := syncStorageCaches(ctx, userCred, provider, &localStorages[i], remoteStorages[i])
				if cachePair.remote != nil && cachePair.local != nil {
					newCacheIds = append(newCacheIds, cachePair)
				}
			}
			if !remoteStorages[i].DisableSync() {
				syncStorageDisks(ctx, userCred, syncResults, provider, driver, &localStorages[i], remoteStorages[i], syncRange)
			}
		}()
	}
	return newCacheIds
}

func syncStorageCaches(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localStorage *SStorage, remoteStorage cloudprovider.ICloudStorage) (cachePair sStoragecacheSyncPair) {
	log.Debugf("syncStorageCaches for storage %s", localStorage.GetId())
	remoteCache := remoteStorage.GetIStoragecache()
	if remoteCache == nil {
		log.Errorf("remote storageCache is nil")
		return
	}
	localCache, isNew, err := StoragecacheManager.SyncWithCloudStoragecache(ctx, userCred, remoteCache, provider)
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
	cachePair.isNew = isNew
	cachePair.region, _ = localStorage.GetRegion()
	return
}

func syncStorageDisks(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localStorage *SStorage, remoteStorage cloudprovider.ICloudStorage, syncRange *SSyncRange) {
	disks, err := func() ([]cloudprovider.ICloudDisk, error) {
		defer syncResults.AddRequestCost(DiskManager)()
		return remoteStorage.GetIDisks()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for storage %s failed %s", remoteStorage.GetName(), err)
		log.Errorf(msg)
		return
	}
	_, _, result := func() ([]SDisk, []cloudprovider.ICloudDisk, compare.SyncResult) {
		defer syncResults.AddSqlCost(DiskManager)()
		return DiskManager.SyncDisks(ctx, userCred, driver, localStorage, disks, provider.GetOwnerId())
	}()

	syncResults.Add(DiskManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncDisks for storage %s result: %s", localStorage.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncZoneHosts(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localZone *SZone, remoteZone cloudprovider.ICloudZone, syncRange *SSyncRange, storageCachePairs []sStoragecacheSyncPair) []sStoragecacheSyncPair {
	hosts, err := func() ([]cloudprovider.ICloudHost, error) {
		defer syncResults.AddRequestCost(HostManager)()
		return remoteZone.GetIHosts()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIHosts for zone %s failed %s", remoteZone.GetName(), err)
		log.Errorf(msg)
		return nil
	}
	localHosts, remoteHosts, result := func() ([]SHost, []cloudprovider.ICloudHost, compare.SyncResult) {
		defer syncResults.AddSqlCost(HostManager)()
		return HostManager.SyncHosts(ctx, userCred, provider, localZone, hosts)
	}()

	syncResults.Add(HostManager, result)

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

			if localHosts[i].Deleted {
				return
			}

			syncMetadata(ctx, userCred, &localHosts[i], remoteHosts[i])
			newCachePairs = syncHostStorages(ctx, userCred, syncResults, provider, &localHosts[i], remoteHosts[i], storageCachePairs)
			syncHostWires(ctx, userCred, syncResults, provider, &localHosts[i], remoteHosts[i])
			syncHostVMs(ctx, userCred, syncResults, provider, driver, &localHosts[i], remoteHosts[i], syncRange)
		}()
	}
	return newCachePairs
}

func syncHostStorages(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localHost *SHost, remoteHost cloudprovider.ICloudHost, storageCachePairs []sStoragecacheSyncPair) []sStoragecacheSyncPair {
	storages, err := func() ([]cloudprovider.ICloudStorage, error) {
		defer syncResults.AddRequestCost(HoststorageManager)()
		return remoteHost.GetIStorages()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		return nil
	}
	localStorages, remoteStorages, result := func() ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
		defer syncResults.AddSqlCost(HoststorageManager)()
		return localHost.SyncHostStorages(ctx, userCred, storages, provider)
	}()

	syncResults.Add(HoststorageManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncHostStorages for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return nil
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)

	newCacheIds := make([]sStoragecacheSyncPair, 0)
	for i := 0; i < len(localStorages); i += 1 {
		syncMetadata(ctx, userCred, &localStorages[i], remoteStorages[i])
		if !isInCache(storageCachePairs, localStorages[i].StoragecacheId) && !isInCache(newCacheIds, localStorages[i].StoragecacheId) {
			cachePair := syncStorageCaches(ctx, userCred, provider, &localStorages[i], remoteStorages[i])
			if cachePair.remote != nil && cachePair.local != nil {
				newCacheIds = append(newCacheIds, cachePair)
			}
		}
	}
	return newCacheIds
}

func syncHostWires(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localHost *SHost, remoteHost cloudprovider.ICloudHost) {
	wires, err := func() ([]cloudprovider.ICloudWire, error) {
		defer func() {
			if syncResults != nil {
				syncResults.AddRequestCost(HostwireManager)()
			}
		}()
		return remoteHost.GetIWires()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIWires for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := func() compare.SyncResult {
		defer func() {
			if syncResults != nil {
				syncResults.AddSqlCost(HostwireManager)()
			}
		}()
		return localHost.SyncHostWires(ctx, userCred, wires)
	}()

	if syncResults != nil {
		syncResults.Add(HostwireManager, result)
	}

	msg := result.Result()
	notes := fmt.Sprintf("SyncHostWires for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.GetParams()), notes, task.GetUserCred(), true)
}

func syncHostVMs(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, driver cloudprovider.ICloudProvider, localHost *SHost, remoteHost cloudprovider.ICloudHost, syncRange *SSyncRange) {
	vms, err := func() ([]cloudprovider.ICloudVM, error) {
		defer syncResults.AddRequestCost(GuestManager)()
		return remoteHost.GetIVMs()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIVMs for host %s failed %s", remoteHost.GetName(), err)
		log.Errorf(msg)
		return
	}

	syncVMPairs, result := func() ([]SGuestSyncResult, compare.SyncResult) {
		defer syncResults.AddSqlCost(GuestManager)()
		return localHost.SyncHostVMs(ctx, userCred, driver, vms, provider.GetOwnerId())
	}()

	syncResults.Add(GuestManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncHostVMs for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	/*if result.IsError() {
		return
	}*/

	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	if result.IsError() {
		logclient.AddSimpleActionLog(provider, logclient.ACT_CLOUD_SYNC, notes, userCred, false)
	}
	for i := 0; i < len(syncVMPairs); i += 1 {
		if !syncVMPairs[i].IsNew && !syncRange.DeepSync {
			continue
		}
		func() {
			lockman.LockObject(ctx, syncVMPairs[i].Local)
			defer lockman.ReleaseObject(ctx, syncVMPairs[i].Local)

			if syncVMPairs[i].Local.Deleted || syncVMPairs[i].Local.PendingDeleted {
				return
			}

			syncVMPeripherals(ctx, userCred, syncVMPairs[i].Local, syncVMPairs[i].Remote, localHost, provider, driver)
		}()
	}

}

func syncVMPeripherals(ctx context.Context, userCred mcclient.TokenCredential, local *SGuest, remote cloudprovider.ICloudVM, host *SHost, provider *SCloudprovider, driver cloudprovider.ICloudProvider) {
	err := syncVMNics(ctx, userCred, provider, host, local, remote)
	if err != nil {
		log.Errorf("syncVMNics error %s", err)
	}
	err = syncVMDisks(ctx, userCred, provider, driver, host, local, remote)
	if err != nil {
		log.Errorf("syncVMDisks error %s", err)
	}
	err = syncVMEip(ctx, userCred, provider, local, remote)
	if err != nil {
		log.Errorf("syncVMEip error %s", err)
	}
	err = syncVMSecgroups(ctx, userCred, provider, local, remote)
	if err != nil {
		log.Errorf("syncVMSecgroups error %s", err)
	}
	result := local.SyncInstanceSnapshots(ctx, userCred, provider)
	if result.IsError() {
		log.Errorf("syncVMInstanceSnapshots error %v", result.AllError())
	}
}

func syncVMNics(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, host *SHost, localVM *SGuest, remoteVM cloudprovider.ICloudVM) error {
	nics, err := remoteVM.GetINics()
	if err != nil {
		// msg := fmt.Sprintf("GetINics for VM %s failed %s", remoteVM.GetName(), err)
		// log.Errorf(msg)
		return errors.Wrap(err, "remoteVM.GetINics")
	}
	result := localVM.SyncVMNics(ctx, userCred, host, nics, nil)
	msg := result.Result()
	notes := fmt.Sprintf("syncVMNics for VM %s result: %s", localVM.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return result.AllError()
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	return nil
}

func syncVMDisks(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, driver cloudprovider.ICloudProvider, host *SHost, localVM *SGuest, remoteVM cloudprovider.ICloudVM) error {
	disks, err := remoteVM.GetIDisks()
	if err != nil {
		// msg := fmt.Sprintf("GetIDisks for VM %s failed %s", remoteVM.GetName(), err)
		// log.Errorf(msg)
		return errors.Wrap(err, "remoteVM.GetIDisks")
	}
	result := localVM.SyncVMDisks(ctx, userCred, driver, host, disks, provider.GetOwnerId())
	msg := result.Result()
	notes := fmt.Sprintf("syncVMDisks for VM %s result: %s", localVM.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return result.AllError()
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
	return nil
}

func syncVMEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localVM *SGuest, remoteVM cloudprovider.ICloudVM) error {
	eip, err := remoteVM.GetIEIP()
	if err != nil {
		// msg := fmt.Sprintf("GetIEIP for VM %s failed %s", remoteVM.GetName(), err)
		// log.Errorf(msg)
		return errors.Wrap(err, "remoteVM.GetIEIP")
	}
	result := localVM.SyncVMEip(ctx, userCred, provider, eip, provider.GetOwnerId())
	msg := result.Result()
	log.Infof("syncVMEip for VM %s result: %s", localVM.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	return nil
}

func syncVMSecgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localVM *SGuest, remoteVM cloudprovider.ICloudVM) error {
	secgroupIds, err := remoteVM.GetSecurityGroupIds()
	if err != nil {
		return errors.Wrap(err, "remoteVM.GetSecurityGroupIds")
	}
	return localVM.SyncVMSecgroups(ctx, userCred, secgroupIds)
}

func syncSkusFromPrivateCloud(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, region *SCloudregion, remoteRegion cloudprovider.ICloudRegion) {
	skus, err := remoteRegion.GetISkus()
	if err != nil {
		msg := fmt.Sprintf("GetISkus for region %s(%s) failed %v", region.Name, region.Id, err)
		log.Errorf(msg)
		return
	}

	result := ServerSkuManager.SyncPrivateCloudSkus(ctx, userCred, region, skus)

	syncResults.Add(ServerSkuManager, result)

	msg := result.Result()
	log.Infof("SyncCloudSkusByRegion for region %s result: %s", region.Name, msg)
	if result.IsError() {
		return
	}
	s := auth.GetSession(ctx, userCred, "")
	if _, err := scheduler.SchedManager.SyncSku(s, true); err != nil {
		log.Errorf("Sync scheduler sku cache error: %v", err)
	}
}

func syncRegionDBInstances(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	instances, err := func() ([]cloudprovider.ICloudDBInstance, error) {
		defer syncResults.AddRequestCost(DBInstanceManager)()
		return remoteRegion.GetIDBInstances()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIDBInstances for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}
	localInstances, remoteInstances, result := func() ([]SDBInstance, []cloudprovider.ICloudDBInstance, compare.SyncResult) {
		defer syncResults.AddSqlCost(DBInstanceManager)()
		return DBInstanceManager.SyncDBInstances(ctx, userCred, provider.GetOwnerId(), provider, localRegion, instances)
	}()

	syncResults.Add(DBInstanceManager, result)
	DBInstanceManager.SyncDBInstanceMasterId(ctx, userCred, provider, instances)

	msg := result.Result()
	log.Infof("SyncDBInstances for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	for i := 0; i < len(localInstances); i++ {
		func() {
			lockman.LockObject(ctx, &localInstances[i])
			defer lockman.ReleaseObject(ctx, &localInstances[i])

			if localInstances[i].Deleted || localInstances[i].PendingDeleted {
				return
			}

			syncDBInstanceResource(ctx, userCred, syncResults, &localInstances[i], remoteInstances[i])
		}()
	}
}

func syncDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	skus, err := func() ([]cloudprovider.ICloudDBInstanceSku, error) {
		defer syncResults.AddRequestCost(DBInstanceSkuManager)()
		return remoteRegion.GetIDBInstanceSkus()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented {
			return
		}
		msg := fmt.Sprintf("GetIDBInstanceSkus for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(DBInstanceSkuManager)()
		return localRegion.SyncDBInstanceSkus(ctx, userCred, provider, skus)
	}()

	syncResults.Add(DBInstanceSkuManager, result)

	msg := result.Result()
	log.Infof("sync rds sku for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncNATSkus(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	skus, err := func() ([]cloudprovider.ICloudNatSku, error) {
		defer syncResults.AddRequestCost(NatSkuManager)()
		return remoteRegion.GetICloudNatSkus()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented {
			return
		}
		msg := fmt.Sprintf("GetINatSkus for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(DBInstanceSkuManager)()
		return localRegion.SyncPrivateCloudNatSkus(ctx, userCred, skus)
	}()

	syncResults.Add(NasSkuManager, result)

	msg := result.Result()
	log.Infof("SyncNasSkus for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncCacheSkus(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	skus, err := func() ([]cloudprovider.ICloudElasticcacheSku, error) {
		defer syncResults.AddRequestCost(ElasticcacheSkuManager)()
		return remoteRegion.GetIElasticcacheSkus()
	}()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented {
			return
		}
		msg := fmt.Sprintf("GetIElasticcacheSkus for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(ElasticcacheSkuManager)()
		return localRegion.SyncPrivateCloudCacheSkus(ctx, userCred, skus)
	}()

	syncResults.Add(NasSkuManager, result)

	msg := result.Result()
	log.Infof("SyncRedisSkus for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncDBInstanceResource(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SDBInstance, remoteInstance cloudprovider.ICloudDBInstance) {
	err := syncDBInstanceNetwork(ctx, userCred, syncResults, localInstance, remoteInstance)
	if err != nil {
		log.Errorf("syncDBInstanceNetwork error: %v", err)
	}
	err = syncDBInstanceSecgroups(ctx, userCred, syncResults, localInstance, remoteInstance)
	if err != nil {
		log.Errorf("syncDBInstanceSecgroups error: %v", err)
	}
	err = syncDBInstanceParameters(ctx, userCred, syncResults, localInstance, remoteInstance)
	if err != nil {
		log.Errorf("syncDBInstanceParameters error: %v", err)
	}
	err = syncDBInstanceDatabases(ctx, userCred, syncResults, localInstance, remoteInstance)
	if err != nil {
		log.Errorf("syncDBInstanceParameters error: %v", err)
	}
	err = syncDBInstanceAccounts(ctx, userCred, syncResults, localInstance, remoteInstance)
	if err != nil {
		log.Errorf("syncDBInstanceAccounts: %v", err)
	}
	err = syncDBInstanceBackups(ctx, userCred, syncResults, localInstance, remoteInstance)
	if err != nil {
		log.Errorf("syncDBInstanceBackups: %v", err)
	}
}

func syncDBInstanceNetwork(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SDBInstance, remoteInstance cloudprovider.ICloudDBInstance) error {
	networks, err := remoteInstance.GetDBNetworks()
	if err != nil {
		return errors.Wrapf(err, "GetDBNetworks")
	}

	result := DBInstanceNetworkManager.SyncDBInstanceNetwork(ctx, userCred, localInstance, networks)
	syncResults.Add(DBInstanceNetworkManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceNetwork for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncDBInstanceSecgroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SDBInstance, remoteInstance cloudprovider.ICloudDBInstance) error {
	secIds, err := remoteInstance.GetSecurityGroupIds()
	if err != nil {
		return errors.Wrapf(err, "GetSecurityGroupIds")
	}
	result := DBInstanceSecgroupManager.SyncDBInstanceSecgroups(ctx, userCred, localInstance, secIds)
	syncResults.Add(DBInstanceSecgroupManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceSecgroups for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncDBInstanceBackups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SDBInstance, remoteInstance cloudprovider.ICloudDBInstance) error {
	backups, err := remoteInstance.GetIDBInstanceBackups()
	if err != nil {
		return errors.Wrapf(err, "GetIDBInstanceBackups")
	}

	region, err := localInstance.GetRegion()
	if err != nil {
		return errors.Wrapf(err, "GetRegion")
	}
	provider := localInstance.GetCloudprovider()

	result := DBInstanceBackupManager.SyncDBInstanceBackups(ctx, userCred, provider, localInstance, region, backups)
	syncResults.Add(DBInstanceBackupManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceBackups for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncDBInstanceParameters(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SDBInstance, remoteInstance cloudprovider.ICloudDBInstance) error {
	parameters, err := remoteInstance.GetIDBInstanceParameters()
	if err != nil {
		return errors.Wrapf(err, "GetIDBInstanceParameters")
	}

	result := DBInstanceParameterManager.SyncDBInstanceParameters(ctx, userCred, localInstance, parameters)
	syncResults.Add(DBInstanceParameterManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceParameters for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncRegionDBInstanceBackups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) error {
	backups, err := func() ([]cloudprovider.ICloudDBInstanceBackup, error) {
		defer syncResults.AddRequestCost(DBInstanceBackupManager)()
		return remoteRegion.GetIDBInstanceBackups()
	}()
	if err != nil {
		return errors.Wrapf(err, "GetIDBInstanceBackups")
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(DBInstanceBackupManager)()
		return DBInstanceBackupManager.SyncDBInstanceBackups(ctx, userCred, provider, nil, localRegion, backups)
	}()

	syncResults.Add(DBInstanceBackupManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceBackups for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncDBInstanceDatabases(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SDBInstance, remoteInstance cloudprovider.ICloudDBInstance) error {
	databases, err := remoteInstance.GetIDBInstanceDatabases()
	if err != nil {
		return errors.Wrapf(err, "GetIDBInstanceDatabases")
	}

	result := DBInstanceDatabaseManager.SyncDBInstanceDatabases(ctx, userCred, localInstance, databases)
	syncResults.Add(DBInstanceDatabaseManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceDatabases for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncDBInstanceAccounts(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SDBInstance, remoteInstance cloudprovider.ICloudDBInstance) error {
	accounts, err := remoteInstance.GetIDBInstanceAccounts()
	if err != nil {
		return errors.Wrapf(err, "GetIDBInstanceAccounts")
	}

	localAccounts, remoteAccounts, result := DBInstanceAccountManager.SyncDBInstanceAccounts(ctx, userCred, localInstance, accounts)
	syncResults.Add(DBInstanceDatabaseManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceAccounts for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return result.AllError()
	}

	for i := 0; i < len(localAccounts); i++ {
		func() {
			lockman.LockObject(ctx, &localAccounts[i])
			defer lockman.ReleaseObject(ctx, &localAccounts[i])

			if localAccounts[i].Deleted {
				return
			}

			err = syncDBInstanceAccountPrivileges(ctx, userCred, syncResults, &localAccounts[i], remoteAccounts[i])
			if err != nil {
				log.Errorf("syncDBInstanceAccountPrivileges error: %v", err)
			}

		}()
	}
	return nil
}

func syncDBInstanceAccountPrivileges(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localAccount *SDBInstanceAccount, remoteAccount cloudprovider.ICloudDBInstanceAccount) error {
	privileges, err := remoteAccount.GetIDBInstanceAccountPrivileges()
	if err != nil {
		return errors.Wrapf(err, "GetIDBInstanceAccountPrivileges for %s(%s)", localAccount.Name, localAccount.Id)
	}

	result := DBInstancePrivilegeManager.SyncDBInstanceAccountPrivileges(ctx, userCred, localAccount, privileges)
	syncResults.Add(DBInstancePrivilegeManager, result)

	msg := result.Result()
	log.Infof("SyncDBInstanceAccountPrivileges for account %s result: %s", localAccount.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncWafIPSets(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	ipSets, err := func() ([]cloudprovider.ICloudWafIPSet, error) {
		defer syncResults.AddRequestCost(WafIPSetManager)()
		return remoteRegion.GetICloudWafIPSets()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudWafIPSets for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(WafIPSetManager)()
		return localRegion.SyncWafIPSets(ctx, userCred, provider, ipSets)
	}()

	syncResults.Add(WafIPSetManager, result)
	log.Infof("SyncWafIPSets for region %s result: %s", localRegion.Name, result.Result())
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncWafRegexSets(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	rSets, err := func() ([]cloudprovider.ICloudWafRegexSet, error) {
		defer syncResults.AddRequestCost(WafRegexSetManager)()
		return remoteRegion.GetICloudWafRegexSets()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudWafRegexSets for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(WafRegexSetManager)()
		return localRegion.SyncWafRegexSets(ctx, userCred, provider, rSets)
	}()
	syncResults.Add(WafRegexSetManager, result)
	log.Infof("SyncWafRegexSets for region %s result: %s", localRegion.Name, result.Result())
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncMongoDBs(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	dbs, err := func() ([]cloudprovider.ICloudMongoDB, error) {
		defer syncResults.AddRequestCost(MongoDBManager)()
		return remoteRegion.GetICloudMongoDBs()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudMongoDBs for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}

	_, _, result := func() ([]SMongoDB, []cloudprovider.ICloudMongoDB, compare.SyncResult) {
		defer syncResults.AddSqlCost(MongoDBManager)()
		return localRegion.SyncMongoDBs(ctx, userCred, provider, dbs)
	}()
	syncResults.Add(MongoDBManager, result)
	msg := result.Result()
	log.Infof("SyncMongoDBs for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}

	return nil
}

func syncElasticSearchs(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	iEss, err := func() ([]cloudprovider.ICloudElasticSearch, error) {
		defer syncResults.AddRequestCost(ElasticSearchManager)()
		return remoteRegion.GetIElasticSearchs()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticSearchs for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(ElasticSearchManager)()
		return localRegion.SyncElasticSearchs(ctx, userCred, provider, iEss)
	}()
	syncResults.Add(ElasticSearchManager, result)
	msg := result.Result()
	log.Infof("SyncElasticSearchs for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncKafkas(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	iKafkas, err := func() ([]cloudprovider.ICloudKafka, error) {
		defer syncResults.AddRequestCost(KafkaManager)()
		return remoteRegion.GetICloudKafkas()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudKafkas for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(KafkaManager)()
		return localRegion.SyncKafkas(ctx, userCred, provider, iKafkas)
	}()
	syncResults.Add(KafkaManager, result)
	msg := result.Result()
	log.Infof("SyncKafkas for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncApps(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	iApps, err := func() ([]cloudprovider.ICloudApp, error) {
		defer syncResults.AddRequestCost(AppManager)()
		return remoteRegion.GetICloudApps()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudApps for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(AppManager)()
		return localRegion.SyncApps(ctx, userCred, provider, iApps)
	}()
	syncResults.Add(AppManager, result)
	msg := result.Result()
	log.Infof("SyncApps for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncKubeClusters(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	iClusters, err := func() ([]cloudprovider.ICloudKubeCluster, error) {
		defer syncResults.AddRequestCost(KubeClusterManager)()
		return remoteRegion.GetICloudKubeClusters()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudKubeClusters for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}
	localClusters, remoteClusters, result := func() ([]SKubeCluster, []cloudprovider.ICloudKubeCluster, compare.SyncResult) {
		defer syncResults.AddSqlCost(KubeClusterManager)()
		return localRegion.SyncKubeClusters(ctx, userCred, provider, iClusters)
	}()
	syncResults.Add(KubeClusterManager, result)
	msg := result.Result()
	log.Infof("SyncKubeClusters for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}

	for i := 0; i < len(localClusters); i++ {
		func() {
			lockman.LockObject(ctx, &localClusters[i])
			defer lockman.ReleaseObject(ctx, &localClusters[i])

			if localClusters[i].Deleted {
				return
			}

			if err := syncKubeClusterNodePools(ctx, userCred, syncResults, &localClusters[i], remoteClusters[i]); err != nil {
				log.Errorf("syncKubeClusterNodePools for %s error: %v", localClusters[i].Name, err)
			}

			if err := syncKubeClusterNodes(ctx, userCred, syncResults, &localClusters[i], remoteClusters[i]); err != nil {
				log.Errorf("syncKubeClusterNodes for %s error: %v", localClusters[i].Name, err)
			}

			if err := localClusters[i].ImportOrUpdate(ctx, userCred, remoteClusters[i]); err != nil {
				log.Errorf("Import cluster %s error: %v", localClusters[i].Name, err)
			}
		}()
	}

	return nil
}

func syncKubeClusterNodePools(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, cluster *SKubeCluster, ext cloudprovider.ICloudKubeCluster) error {
	iPools, err := func() ([]cloudprovider.ICloudKubeNodePool, error) {
		defer syncResults.AddRequestCost(KubeNodePoolManager)()
		return ext.GetIKubeNodePools()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudKubeNodePools for cluster %s failed %s", cluster.GetName(), err)
		log.Errorf(msg)
		return err
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(KubeNodePoolManager)()
		return cluster.SyncKubeNodePools(ctx, userCred, iPools)
	}()
	syncResults.Add(KubeNodePoolManager, result)
	msg := result.Result()
	log.Infof("SyncKubeNodePools for cluster %s result: %s", cluster.Name, msg)
	if result.IsError() {
		return result.AllError()
	}

	return nil
}

func syncKubeClusterNodes(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, cluster *SKubeCluster, ext cloudprovider.ICloudKubeCluster) error {
	iNodes, err := func() ([]cloudprovider.ICloudKubeNode, error) {
		defer syncResults.AddRequestCost(KubeNodeManager)()
		return ext.GetIKubeNodes()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudKubeNodes for cluster %s failed %s", cluster.GetName(), err)
		log.Errorf(msg)
		return err
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(KubeNodeManager)()
		return cluster.SyncKubeNodes(ctx, userCred, iNodes)
	}()
	syncResults.Add(KubeNodeManager, result)
	msg := result.Result()
	log.Infof("SyncKubeNodes for cluster %s result: %s", cluster.Name, msg)
	if result.IsError() {
		return result.AllError()
	}

	return nil
}

func syncWafInstances(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	wafIns, err := func() ([]cloudprovider.ICloudWafInstance, error) {
		defer syncResults.AddRequestCost(WafInstanceManager)()
		return remoteRegion.GetICloudWafInstances()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudWafInstances for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}

	localWafs, remoteWafs, result := func() ([]SWafInstance, []cloudprovider.ICloudWafInstance, compare.SyncResult) {
		defer syncResults.AddSqlCost(WafInstanceManager)()
		return localRegion.SyncWafInstances(ctx, userCred, provider, wafIns)
	}()
	syncResults.Add(WafInstanceManager, result)
	msg := result.Result()
	log.Infof("SyncWafInstances for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}

	for i := 0; i < len(localWafs); i++ {
		func() {
			lockman.LockObject(ctx, &localWafs[i])
			defer lockman.ReleaseObject(ctx, &localWafs[i])

			if localWafs[i].Deleted {
				return
			}

			err = syncWafRules(ctx, userCred, syncResults, &localWafs[i], remoteWafs[i])
			if err != nil {
				log.Errorf("syncDBInstanceAccountPrivileges error: %v", err)
			}

		}()
	}

	return nil
}

func syncWafRules(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localWaf *SWafInstance, remoteWafs cloudprovider.ICloudWafInstance) error {
	rules, err := func() ([]cloudprovider.ICloudWafRule, error) {
		defer syncResults.AddRequestCost(WafRuleManager)()
		return remoteWafs.GetRules()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetRules for waf instance %s failed %s", localWaf.Name, err)
		log.Errorf(msg)
		return err
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(WafRuleManager)()
		return localWaf.SyncWafRules(ctx, userCred, rules)
	}()
	syncResults.Add(WafRuleManager, result)
	msg := result.Result()
	log.Infof("SyncWafRules for waf %s result: %s", localWaf.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncRegionSnapshots(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	snapshots, err := func() ([]cloudprovider.ICloudSnapshot, error) {
		defer syncResults.AddRequestCost(SnapshotManager)()
		return remoteRegion.GetISnapshots()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetISnapshots for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(SnapshotManager)()
		return SnapshotManager.SyncSnapshots(ctx, userCred, provider, localRegion, snapshots, provider.GetOwnerId())
	}()

	syncResults.Add(SnapshotManager, result)

	msg := result.Result()
	log.Infof("SyncSnapshots for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
}

func syncRegionSnapshotPolicies(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	snapshotPolicies, err := func() ([]cloudprovider.ICloudSnapshotPolicy, error) {
		defer syncResults.AddRequestCost(SnapshotPolicyManager)()
		return remoteRegion.GetISnapshotPolicies()
	}()
	if err != nil {
		log.Errorf("GetISnapshotPolicies for region %s failed %s", remoteRegion.GetName(), err)
		return
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(SnapshotPolicyManager)()
		return SnapshotPolicyManager.SyncSnapshotPolicies(ctx, userCred, provider, localRegion, snapshotPolicies, provider.GetOwnerId())
	}()
	syncResults.Add(SnapshotPolicyManager, result)
	msg := result.Result()
	log.Infof("SyncSnapshotPolicies for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionNetworkInterfaces(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	networkInterfaces, err := func() ([]cloudprovider.ICloudNetworkInterface, error) {
		defer syncResults.AddRequestCost(NetworkInterfaceManager)()
		return remoteRegion.GetINetworkInterfaces()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetINetworkInterfaces for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	localInterfaces, remoteInterfaces, result := func() ([]SNetworkInterface, []cloudprovider.ICloudNetworkInterface, compare.SyncResult) {
		defer syncResults.AddSqlCost(NetworkInterfaceManager)()
		return NetworkInterfaceManager.SyncNetworkInterfaces(ctx, userCred, provider, localRegion, networkInterfaces)
	}()
	syncResults.Add(NetworkInterfaceManager, result)

	msg := result.Result()
	log.Infof("SyncNetworkInterfaces for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}

	for i := 0; i < len(localInterfaces); i++ {
		func() {
			lockman.LockObject(ctx, &localInterfaces[i])
			defer lockman.ReleaseObject(ctx, &localInterfaces[i])

			if localInterfaces[i].Deleted {
				return
			}

			syncInterfaceAddresses(ctx, userCred, &localInterfaces[i], remoteInterfaces[i])
		}()
	}
}

func syncInterfaceAddresses(ctx context.Context, userCred mcclient.TokenCredential, localInterface *SNetworkInterface, remoteInterface cloudprovider.ICloudNetworkInterface) {
	addresses, err := remoteInterface.GetICloudInterfaceAddresses()
	if err != nil {
		msg := fmt.Sprintf("GetICloudInterfaceAddresses for networkinterface %s failed %s", remoteInterface.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := NetworkinterfacenetworkManager.SyncInterfaceAddresses(ctx, userCred, localInterface, addresses)
	msg := result.Result()
	notes := fmt.Sprintf("SyncInterfaceAddresses for networkinterface %s result: %s", localInterface.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
}

func syncPublicCloudProviderInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	syncResults SSyncResultSet,
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

	if cloudprovider.IsSupportQuota(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_QUOTA) {
		syncRegionQuotas(ctx, userCred, syncResults, driver, provider, localRegion, remoteRegion)
	}

	localZones, remoteZones, _ := syncRegionZones(ctx, userCred, syncResults, provider, localRegion, remoteRegion)

	if !driver.GetFactory().NeedSyncSkuFromCloud() {
		syncRegionSkus(ctx, userCred, localRegion)
		SyncRegionDBInstanceSkus(ctx, userCred, localRegion.Id, true)
		SyncRegionNatSkus(ctx, userCred, localRegion.Id, true)
		SyncRegionNasSkus(ctx, userCred, localRegion.Id, true)
	} else {
		syncSkusFromPrivateCloud(ctx, userCred, syncResults, localRegion, remoteRegion)
		if cloudprovider.IsSupportRds(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_RDS) {
			syncDBInstanceSkus(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		}
		if cloudprovider.IsSupportNAT(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_NAT) {
			syncNATSkus(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		}
		if cloudprovider.IsSupportElasticCache(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_CACHE) {
			syncCacheSkus(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		}
	}

	// no need to lock public cloud region as cloud region for public cloud is readonly

	if cloudprovider.IsSupportObjectstore(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE) {
		syncRegionBuckets(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportCompute(driver) {
		if syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_NETWORK) || syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_EIP) {
			// 需要先同步vpc，避免私有云eip找不到network
			if !(driver.GetFactory().IsPublicCloud() && !syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_NETWORK)) {
				syncRegionVPCs(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
			}
			syncRegionEips(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		}

		if syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_COMPUTE) {
			// sync snapshot policies before sync disks
			syncRegionSnapshotPolicies(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)

			for j := 0; j < len(localZones); j += 1 {

				if len(syncRange.Zone) > 0 && !utils.IsInStringArray(localZones[j].Id, syncRange.Zone) {
					continue
				}
				// no need to lock zone as public cloud zone is read-only

				newPairs := syncZoneStorages(ctx, userCred, syncResults, provider, driver, &localZones[j], remoteZones[j], syncRange, storageCachePairs)
				if len(newPairs) > 0 {
					storageCachePairs = append(storageCachePairs, newPairs...)
				}
				newPairs = syncZoneHosts(ctx, userCred, syncResults, provider, driver, &localZones[j], remoteZones[j], syncRange, storageCachePairs)
				if len(newPairs) > 0 {
					storageCachePairs = append(storageCachePairs, newPairs...)
				}
			}

			// sync snapshots after sync disks
			syncRegionSnapshots(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		}
	}

	if cloudprovider.IsSupportNAS(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_NAS) {
		syncRegionAccessGroups(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		syncRegionFileSystems(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
	}

	if cloudprovider.IsSupportLoadbalancer(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_LOADBALANCER) {
		syncRegionLoadbalancerAcls(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		syncRegionLoadbalancerCertificates(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		syncRegionLoadbalancers(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
	}

	if cloudprovider.IsSupportCompute(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_COMPUTE) {
		syncRegionNetworkInterfaces(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
	}

	if cloudprovider.IsSupportRds(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_RDS) {
		syncRegionDBInstances(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
		syncRegionDBInstanceBackups(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
	}

	if cloudprovider.IsSupportElasticCache(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_CACHE) {
		syncElasticcaches(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)
	}

	if cloudprovider.IsSupportWaf(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_WAF) {
		syncWafIPSets(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
		syncWafRegexSets(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
		syncWafInstances(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportMongoDB(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_MONGO_DB) {
		syncMongoDBs(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportElasticSearch(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_ES) {
		syncElasticSearchs(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportKafka(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_KAFKA) {
		syncKafkas(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportApp(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_APP) {
		syncApps(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportContainer(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_CONTAINER) {
		syncKubeClusters(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportTablestore(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_TABLESTORE) {
		syncTablestore(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportCompute(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_COMPUTE) {
		log.Debugf("storageCachePairs count %d", len(storageCachePairs))
		for i := range storageCachePairs {
			// always sync private cloud cached images
			if storageCachePairs[i].isNew || syncRange.DeepSync || !driver.GetFactory().IsPublicCloud() {
				result := func() compare.SyncResult {
					defer syncResults.AddRequestCost(CachedimageManager)()
					return storageCachePairs[i].syncCloudImages(ctx, userCred)
				}()

				syncResults.Add(CachedimageManager, result)

				msg := result.Result()
				log.Infof("syncCloudImages result: %s", msg)
			}
		}
	}

	if cloudprovider.IsSupportModelartsPool(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_MODELARTES) {
		syncModelartsPoolSkus(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
		syncModelartsPools(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	if cloudprovider.IsSupportMiscResources(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_MISC) {
		syncMiscResources(ctx, userCred, syncResults, provider, localRegion, remoteRegion)
	}

	return nil
}

func getZoneForPremiseCloudRegion(ctx context.Context, userCred mcclient.TokenCredential, iregion cloudprovider.ICloudRegion) (*SZone, error) {
	extHosts, err := iregion.GetIHosts()
	if err != nil {
		return nil, errors.Wrap(err, "unable to GetIHosts")
	}
	for _, extHost := range extHosts {
		// onpremise host
		accessIp := extHost.GetAccessIp()
		if len(accessIp) == 0 {
			msg := fmt.Sprintf("fail to find wire for host %s: empty host access ip", extHost.GetName())
			log.Errorf(msg)
			continue
		}
		wire, err := WireManager.GetOnPremiseWireOfIp(accessIp)
		if err != nil {
			msg := fmt.Sprintf("fail to find wire for host %s %s: %s", extHost.GetName(), accessIp, err)
			log.Errorf(msg)
			continue
		}
		return wire.GetZone()
	}
	return nil, errors.Wrap(errors.ErrNotFound, "no suitable zone")
}

func syncOnPremiseCloudProviderStorage(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, iregion cloudprovider.ICloudRegion, driver cloudprovider.ICloudProvider, syncRange *SSyncRange) []sStoragecacheSyncPair {
	istorages, err := iregion.GetIStorages()
	if err != nil {
		msg := fmt.Sprintf("GetIStorages for provider %s failed %s", provider.GetName(), err)
		log.Errorf(msg)
		return nil
	}
	zone, err := getZoneForPremiseCloudRegion(ctx, userCred, iregion)
	if err != nil {
		msg := fmt.Sprintf("Can't get zone for Premise cloud region %s", iregion.GetName())
		log.Errorf(msg)
		return nil
	}
	localStorages, remoteStorages, result := StorageManager.SyncStorages(ctx, userCred, provider, zone, istorages)
	syncResults.Add(StorageManager, result)

	msg := result.Result()
	notes := fmt.Sprintf("SyncStorages for provider %s result: %s", provider.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return nil
	}

	storageCachePairs := make([]sStoragecacheSyncPair, 0)
	for i := 0; i < len(localStorages); i += 1 {
		func() {
			lockman.LockObject(ctx, &localStorages[i])
			defer lockman.ReleaseObject(ctx, &localStorages[i])

			if localStorages[i].Deleted {
				return
			}

			if !isInCache(storageCachePairs, localStorages[i].StoragecacheId) {
				cachePair := syncStorageCaches(ctx, userCred, provider, &localStorages[i], remoteStorages[i])
				if cachePair.remote != nil && cachePair.local != nil {
					storageCachePairs = append(storageCachePairs, cachePair)
				}
			}
			if !remoteStorages[i].DisableSync() {
				syncStorageDisks(ctx, userCred, syncResults, provider, driver, &localStorages[i], remoteStorages[i], syncRange)
			}
		}()
	}
	return storageCachePairs
}

func syncOnPremiseCloudProviderInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	syncResults SSyncResultSet,
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

	localRegion := CloudregionManager.FetchDefaultRegion()

	if cloudprovider.IsSupportObjectstore(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE) {
		syncRegionBuckets(ctx, userCred, syncResults, provider, localRegion, iregion)
	}

	var storageCachePairs []sStoragecacheSyncPair
	if cloudprovider.IsSupportCompute(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_COMPUTE) {
		storageCachePairs = syncOnPremiseCloudProviderStorage(ctx, userCred, syncResults, provider, iregion, driver, syncRange)
		ihosts, err := func() ([]cloudprovider.ICloudHost, error) {
			defer syncResults.AddRequestCost(HostManager)()
			return iregion.GetIHosts()
		}()
		if err != nil {
			msg := fmt.Sprintf("GetIHosts for provider %s failed %s", provider.GetName(), err)
			log.Errorf(msg)
			return err
		}

		localHosts, remoteHosts, result := func() ([]SHost, []cloudprovider.ICloudHost, compare.SyncResult) {
			defer syncResults.AddSqlCost(HostManager)()
			return HostManager.SyncHosts(ctx, userCred, provider, nil, ihosts)
		}()

		syncResults.Add(HostManager, result)

		msg := result.Result()
		notes := fmt.Sprintf("SyncHosts for provider %s result: %s", provider.Name, msg)
		log.Infof(notes)

		for i := 0; i < len(localHosts); i += 1 {
			if len(syncRange.Host) > 0 && !utils.IsInStringArray(localHosts[i].Id, syncRange.Host) {
				continue
			}
			newCachePairs := syncHostStorages(ctx, userCred, syncResults, provider, &localHosts[i], remoteHosts[i], storageCachePairs)
			if len(newCachePairs) > 0 {
				storageCachePairs = append(storageCachePairs, newCachePairs...)
			}
			syncHostNics(ctx, userCred, provider, &localHosts[i], remoteHosts[i])
			syncOnPremiseHostWires(ctx, userCred, syncResults, provider, &localHosts[i], remoteHosts[i])
			syncHostVMs(ctx, userCred, syncResults, provider, driver, &localHosts[i], remoteHosts[i], syncRange)
		}
	}

	if cloudprovider.IsSupportCompute(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_COMPUTE) {
		log.Debugf("storageCachePairs count %d", len(storageCachePairs))
		for i := range storageCachePairs {
			// alway sync on-premise cached images
			// if storageCachePairs[i].isNew || syncRange.DeepSync {
			result := func() compare.SyncResult {
				defer syncResults.AddRequestCost(CachedimageManager)()
				return storageCachePairs[i].syncCloudImages(ctx, userCred)
			}()

			syncResults.Add(CachedimageManager, result)
			msg := result.Result()
			log.Infof("syncCloudImages for stroagecache %s result: %s", storageCachePairs[i].local.GetId(), msg)
			// }
		}
	}

	return nil
}

func syncOnPremiseHostWires(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localHost *SHost, remoteHost cloudprovider.ICloudHost) {
	log.Infof("start to sync OnPremeseHostWires")
	if provider.Provider != api.CLOUD_PROVIDER_VMWARE {
		return
	}
	func() {
		defer func() {
			if syncResults != nil {
				syncResults.AddSqlCost(HostwireManager)()
			}
		}()
		result := localHost.SyncEsxiHostWires(ctx, userCred, remoteHost)
		if syncResults != nil {
			syncResults.Add(HostwireManager, result)
		}

		msg := result.Result()
		notes := fmt.Sprintf("SyncEsxiHostWires for host %s result: %s", localHost.Name, msg)
		if result.IsError() {
			log.Errorf(notes)
			return
		} else {
			log.Infof(notes)
		}
		db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
	}()
}

func syncHostNics(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localHost *SHost, remoteHost cloudprovider.ICloudHost) {
	result := localHost.SyncHostExternalNics(ctx, userCred, remoteHost)
	msg := result.Result()
	notes := fmt.Sprintf("SyncHostWires for host %s result: %s", localHost.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		return
	}
	// db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, userCred)
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
			recs[i].SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
	}
}

func SyncCloudProject(userCred mcclient.TokenCredential, model db.IVirtualModel, syncOwnerId mcclient.IIdentityProvider, extModel cloudprovider.IVirtualResource, managerId string) {
	newOwnerId, err := func() (mcclient.IIdentityProvider, error) {
		_manager, err := CloudproviderManager.FetchById(managerId)
		if err != nil {
			return nil, errors.Wrapf(err, "CloudproviderManager.FetchById(%s)", managerId)
		}
		manager := _manager.(*SCloudprovider)
		rm, err := manager.GetProjectMapping()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetProjectMapping")
		}
		account, err := manager.GetCloudaccount()
		if err != nil {
			return nil, errors.Wrapf(err, "GetCloudaccount")
		}
		if rm != nil && rm.Enabled.Bool() {
			extTags, err := extModel.GetTags()
			if err != nil {
				return nil, errors.Wrapf(err, "extModel.GetTags")
			}
			if rm.Rules != nil {
				for _, rule := range *rm.Rules {
					domainId, projectId, newProj, isMatch := rule.IsMatchTags(extTags)
					if isMatch {
						if len(newProj) > 0 {
							domainId, projectId, err = account.getOrCreateTenant(context.TODO(), newProj, "", "", "auto create from tag")
							if err != nil {
								return nil, errors.Wrapf(err, "getOrCreateTenant(%s)", newProj)
							}
						}
						if len(domainId) > 0 && len(projectId) > 0 {
							return &db.SOwnerId{DomainId: domainId, ProjectId: projectId}, nil
						}
					}
				}
			}
		}
		return nil, nil
	}()
	if err != nil {
		log.Errorf("try sync project for %s %s by tags error: %v", model.Keyword(), model.GetName(), err)
	}
	if extProjectId := extModel.GetProjectId(); len(extProjectId) > 0 && newOwnerId == nil {
		extProject, err := ExternalProjectManager.GetProject(extProjectId, managerId)
		if err != nil {
			log.Errorf("sync project for %s %s error: %v", model.Keyword(), model.GetName(), err)
		} else if len(extProject.ProjectId) > 0 {
			newOwnerId = extProject.GetOwnerId()
		}
	}
	if newOwnerId == nil && syncOwnerId != nil && len(syncOwnerId.GetProjectId()) > 0 {
		newOwnerId = syncOwnerId
	}
	if newOwnerId == nil {
		newOwnerId = userCred
	}
	model.SyncCloudProjectId(userCred, newOwnerId)
}

func SyncCloudDomain(userCred mcclient.TokenCredential, model db.IDomainLevelModel, syncOwnerId mcclient.IIdentityProvider) {
	var newOwnerId mcclient.IIdentityProvider
	if syncOwnerId != nil && len(syncOwnerId.GetProjectDomainId()) > 0 {
		newOwnerId = syncOwnerId
	}
	if newOwnerId == nil {
		newOwnerId = userCred
	}
	model.SyncCloudDomainId(userCred, newOwnerId)
}

func SyncCloudaccountResources(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount, syncRange *SSyncRange) error {
	provider, err := account.GetProvider(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}

	if cloudprovider.IsSupportProject(provider) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_PROJECT) {
		err = syncProjects(ctx, userCred, SSyncResultSet{}, account, provider)
		if err != nil {
			log.Errorf("Sync project for account %s error: %v", account.Name, err)
		}
	}

	if cloudprovider.IsSupportDnsZone(provider) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_DNSZONE) {
		err = syncDns(ctx, userCred, SSyncResultSet{}, account, provider)
		if err != nil {
			log.Errorf("Sync dns zone for account %s error: %v", account.Name, err)
		}
	}

	return nil
}

func syncProjects(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, account *SCloudaccount, provider cloudprovider.ICloudProvider) error {
	lockman.LockRawObject(ctx, ExternalProjectManager.Keyword(), account.Id)
	defer lockman.ReleaseRawObject(ctx, ExternalProjectManager.Keyword(), account.Id)

	projects, err := func() ([]cloudprovider.ICloudProject, error) {
		defer syncResults.AddRequestCost(ExternalProjectManager)()
		return provider.GetIProjects()
	}()
	if err != nil {
		return errors.Wrapf(err, "GetIProjects")
	}

	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(ExternalProjectManager)()
		return ExternalProjectManager.SyncProjects(ctx, userCred, account, projects)
	}()

	syncResults.Add(ExternalProjectManager, result)

	msg := result.Result()
	log.Infof("SyncProjects for account %s result: %s", account.Name, msg)
	if result.IsError() {
		return err
	}
	return nil
}

func syncDns(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, account *SCloudaccount, provider cloudprovider.ICloudProvider) error {
	lockman.LockRawObject(ctx, DnsZoneCacheManager.Keyword(), account.Id)
	defer lockman.ReleaseRawObject(ctx, DnsZoneCacheManager.Keyword(), account.Id)

	dnsZones, err := provider.GetICloudDnsZones()
	if err != nil {
		return errors.Wrapf(err, "GetICloudDnsZones")
	}
	localZones, remoteZones, result := account.SyncDnsZones(ctx, userCred, dnsZones)
	log.Infof("Sync dns zones for cloudaccount %s result: %s", account.Name, result.Result())
	for i := 0; i < len(localZones); i++ {
		func() {
			lockman.LockObject(ctx, &localZones[i])
			defer lockman.ReleaseObject(ctx, &localZones[i])

			if localZones[i].Deleted {
				return
			}

			result := localZones[i].SyncDnsRecordSets(ctx, userCred, account.Provider, remoteZones[i])
			log.Infof("Sync dns records for dns zone %s result: %s", localZones[i].GetName(), result.Result())
		}()
	}
	return nil
}

func SyncCloudproviderResources(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, syncRange *SSyncRange) error {
	driver, err := provider.GetProvider(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}

	if cloudprovider.IsSupportCDN(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_CDN) {
		err = syncCdnDomains(ctx, userCred, SSyncResultSet{}, provider, driver)
		if err != nil {
			log.Errorf("syncCdnDomains error: %v", err)
		}
	}

	if cloudprovider.IsSupportInterVpcNetwork(driver) && syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_INTERVPCNETWORK) {
		err = syncInterVpcNetworks(ctx, userCred, SSyncResultSet{}, provider, driver)
		if err != nil {
			log.Errorf("syncInterVpcNetworks error: %v", err)
		}
	}

	if syncRange.NeedSyncResource(cloudprovider.CLOUD_CAPABILITY_NETWORK) {
		err = syncGlobalVpcs(ctx, userCred, SSyncResultSet{}, provider, driver)
		if err != nil {
			log.Errorf("syncGlobalVpcs error: %v", err)
		}
	}

	return nil
}

func syncCdnDomains(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, driver cloudprovider.ICloudProvider) error {
	domains, err := driver.GetICloudCDNDomains()
	if err != nil {
		return err
	}

	result := provider.SyncCDNDomains(ctx, userCred, domains)
	log.Infof("Sync CDN for provider %s result: %s", provider.Name, result.Result())
	return nil
}

func syncInterVpcNetworks(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, driver cloudprovider.ICloudProvider) error {
	networks, err := driver.GetICloudInterVpcNetworks()
	if err != nil {
		return errors.Wrapf(err, "GetICloudInterVpcNetworks")
	}
	localNetwork, remoteNetwork, result := provider.SyncInterVpcNetwork(ctx, userCred, networks)
	log.Infof("Sync inter vpc network for cloudprovider %s result: %s", provider.GetName(), result.Result())
	for i := range localNetwork {
		lockman.LockObject(ctx, &localNetwork[i])
		defer lockman.ReleaseObject(ctx, &localNetwork[i])

		if localNetwork[i].Deleted {
			continue
		}
		localNetwork[i].SyncInterVpcNetworkRouteSets(ctx, userCred, remoteNetwork[i])
	}
	return nil
}

func syncGlobalVpcs(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, driver cloudprovider.ICloudProvider) error {
	gvpcs, err := driver.GetICloudGlobalVpcs()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
			return nil
		}
		return err
	}

	result := provider.SyncGlobalVpcs(ctx, userCred, gvpcs)
	log.Infof("Sync global vpcs for cloudprovider %s result: %s", provider.GetName(), result.Result())
	return nil
}

func syncTablestore(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	iTablestores, err := func() ([]cloudprovider.ICloudTablestore, error) {
		defer syncResults.AddRequestCost(TablestoreManager)()
		return remoteRegion.GetICloudTablestores()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetICloudTablestores for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return err
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(TablestoreManager)()
		return localRegion.SyncTablestores(ctx, userCred, iTablestores, provider)
	}()
	syncResults.Add(TablestoreManager, result)
	msg := result.Result()
	log.Infof("SyncTablestores for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncModelartsPools(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	ipools, err := remoteRegion.GetIModelartsPools()
	if err != nil {
		msg := fmt.Sprintf("GetIModelartsPools for provider %s failed %s", err, ipools)
		log.Errorf(msg)
		return err
	}
	result := localRegion.SyncModelartsPools(ctx, userCred, provider, ipools)
	log.Infof("SyncModelartsPools for region %s result: %s", provider.GetName(), result.Result())
	return nil
}

func syncModelartsPoolSkus(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	ipools, err := remoteRegion.GetIModelartsPoolSku()
	if err != nil {
		msg := fmt.Sprintf("GetIModelartsPoolSku for provider %s failed %s", err, ipools)
		log.Errorf(msg)
		return err
	}
	result := localRegion.SyncModelartsPoolSkus(ctx, userCred, provider, ipools)
	log.Infof("SyncModelartsPoolSkus for region %s result: %s", provider.GetName(), result.Result())
	return nil
}

func syncMiscResources(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion) error {
	exts, err := remoteRegion.GetIMiscResources()
	if err != nil {
		msg := fmt.Sprintf("GetIMiscResources for provider %s failed %v", provider.Name, err)
		log.Errorf(msg)
		return err
	}
	result := localRegion.SyncMiscResources(ctx, userCred, provider, exts)
	log.Infof("SyncMiscResources for provider %s result: %s", provider.GetName(), result.Result())
	return nil
}
