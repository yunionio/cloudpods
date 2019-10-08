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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAwsCachedLbbgManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var AwsCachedLbbgManager *SAwsCachedLbbgManager

func init() {
	AwsCachedLbbgManager = &SAwsCachedLbbgManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAwsCachedLbbg{},
			"awscachedlbbgs_tbl",
			"awscachedlbbg",
			"awscachedlbbgs",
		),
	}
	AwsCachedLbbgManager.SetVirtualObject(AwsCachedLbbgManager)
}

type SAwsCachedLbbg struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	LoadbalancerId      string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendGroupId      string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	TargetType          string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"` // 后端服务器类型
	ProtocolType        string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"` // 监听协议类型
	Port                int    `nullable:"false" list:"user" create:"required"`                            // 监听端口
	HealthCheckProtocol string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"` // 健康检查协议类型
	HealthCheckInterval int    `nullable:"false" list:"user" create:"required"`                            // 健康检查时间间隔
}

func (lbb *SAwsCachedLbbg) GetCustomizeColumns(context.Context, mcclient.TokenCredential, jsonutils.JSONObject) *jsonutils.JSONDict {
	return nil
}

func (lbbg *SAwsCachedLbbg) GetLocalBackendGroup(ctx context.Context, userCred mcclient.TokenCredential) (*SLoadbalancerBackendGroup, error) {
	if len(lbbg.BackendGroupId) == 0 {
		return nil, fmt.Errorf("GetLocalBackendGroup no related local backendgroup")
	}

	locallbbg, err := db.FetchById(LoadbalancerBackendGroupManager, lbbg.BackendGroupId)
	if err != nil {
		return nil, err
	}

	return locallbbg.(*SLoadbalancerBackendGroup), err
}

func (lbbg *SAwsCachedLbbg) GetLoadbalancer() *SLoadbalancer {
	lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
	if err != nil {
		log.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		return nil
	}
	return lb.(*SLoadbalancer)
}

func (lbbg *SAwsCachedLbbg) GetCachedBackends() ([]SAwsCachedLb, error) {
	ret := []SAwsCachedLb{}
	err := AwsCachedLbbgManager.Query().Equals("cached_backend_group_id", lbbg.GetId()).IsFalse("pending_deleted").All(&ret)
	if err != nil {
		log.Errorf("failed to get cached backends for backendgroup %s", lbbg.Name)
		return nil, err
	}

	return ret, nil
}

func (lbbg *SAwsCachedLbbg) GetICloudLoadbalancerBackendGroup() (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	if len(lbbg.ExternalId) == 0 {
		return nil, fmt.Errorf("backendgroup %s has no external id", lbbg.GetId())
	}

	lb := lbbg.GetLoadbalancer()
	if lb == nil {
		return nil, fmt.Errorf("backendgroup %s releated loadbalancer not found", lbbg.GetId())
	}

	iregion, err := lb.GetIRegion()
	if err != nil {
		return nil, err
	}

	ilb, err := iregion.GetILoadBalancerById(lb.GetExternalId())
	if err != nil {
		return nil, err
	}

	ilbbg, err := ilb.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
	if err != nil {
		return nil, err
	}

	return ilbbg, nil
}

func (man *SAwsCachedLbbgManager) GetUsableCachedBackendGroups(loadbalancerId, backendGroupId, protocolType, healthCheckProtocol string, healthCheckInterval int) ([]SAwsCachedLbbg, error) {
	ret := []SAwsCachedLbbg{}
	q := man.Query().Equals("protocol_type", protocolType).IsNotEmpty("external_id").Equals("health_check_protocol", healthCheckProtocol)
	q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("loadbalancer_id"), loadbalancerId), sqlchemy.IsNullOrEmpty(q.Field("loadbalancer_id"))))
	if !utils.IsInStringArray(protocolType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) {
		// healthCheckInterval 10/30
		q = q.Equals("health_check_interval", healthCheckInterval)
	}
	q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("backend_group_id"), backendGroupId), sqlchemy.IsNullOrEmpty(q.Field("backend_group_id"))))
	q = q.IsFalse("pending_deleted")
	err := q.All(&ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func (man *SAwsCachedLbbgManager) GetUsableCachedBackendGroup(loadbalancerId, backendGroupId, protocolType, healthCheckProtocol string, healthCheckInterval int) (*SAwsCachedLbbg, error) {
	ret, err := man.GetUsableCachedBackendGroups(loadbalancerId, backendGroupId, protocolType, healthCheckProtocol, healthCheckInterval)
	if err != nil {
		return nil, err
	}

	if len(ret) > 0 {
		cachedLbbg := &ret[0]
		cachedLbbg.SetModelManager(AwsCachedLbbgManager, cachedLbbg)
		if cachedLbbg.LoadbalancerId == "" {
			_, err := db.Update(cachedLbbg, func() error {
				cachedLbbg.LoadbalancerId = loadbalancerId
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		return cachedLbbg, nil
	}

	return nil, nil
}

func (man *SAwsCachedLbbgManager) GetCachedBackendGroups(backendGroupId string) ([]SAwsCachedLbbg, error) {
	ret := []SAwsCachedLbbg{}
	err := man.Query().IsFalse("pending_deleted").Equals("backend_group_id", backendGroupId).All(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (man *SAwsCachedLbbgManager) getLoadbalancerBackendgroupsByRegion(regionId string) ([]SAwsCachedLbbg, error) {
	lbbgs := []SAwsCachedLbbg{}
	q := man.Query().Equals("cloudregion_id", regionId).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for region: %s error: %v", regionId, err)
		return nil, err
	}
	return lbbgs, nil
}

func (man *SAwsCachedLbbgManager) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, lbbgs []cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) ([]SAwsCachedLbbg, []cloudprovider.ICloudLoadbalancerBackendGroup, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))

	localLbgs := []SAwsCachedLbbg{}
	remoteLbbgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	syncResult := compare.SyncResult{}

	dbLbbgs, err := man.getLoadbalancerBackendgroupsByRegion(region.GetId())
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SAwsCachedLbbg{}
	commondb := []SAwsCachedLbbg{}
	commonext := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	added := []cloudprovider.ICloudLoadbalancerBackendGroup{}

	err = compare.CompareSets(dbLbbgs, lbbgs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerBackendgroup(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		var elb *SLoadbalancer
		elbIds := commonext[i].GetLoadbalancerId()
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		elbId := commonext[i].GetLoadbalancerId()
		if len(elbIds) > 0 {
			ielb, err := db.FetchByExternalId(LoadbalancerManager, elbId)
			if err == nil {
				elb = ielb.(*SLoadbalancer)
			}
		}

		if elb == nil {
			elb = &SLoadbalancer{}
			elb.Id = ""
			elb.CloudregionId = region.GetId()
			elb.ManagerId = provider.GetId()
		}

		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, elb, commonext[i], provider.GetOwnerId())
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localLbgs = append(localLbgs, commondb[i])
			remoteLbbgs = append(remoteLbbgs, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		var elb *SLoadbalancer
		elbId := added[i].GetLoadbalancerId()
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		if len(elbId) > 0 {
			elb, err = LoadbalancerManager.FetchByExternalId(provider.GetId(), elbId)
			if err != nil {
				log.Debugf("awsCachedLbbgManager.SyncLoadbalancerBackendgroups %s", err)
			}
		}

		if elb == nil {
			elb = &SLoadbalancer{}
			elb.Id = ""
			elb.CloudregionId = region.GetId()
			elb.ManagerId = provider.GetId()
		}

		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, elb, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localLbgs = append(localLbgs, *new)
			remoteLbbgs = append(remoteLbbgs, added[i])
			syncResult.Add()
		}
	}
	return localLbgs, remoteLbbgs, syncResult
}

func (lbbg *SAwsCachedLbbg) syncRemoveCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbbg)
	defer lockman.ReleaseObject(ctx, lbbg)

	err := lbbg.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbbg.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		lbbg.SetModelManager(AwsCachedLbbgManager, lbbg)
		err := db.DeleteModel(ctx, userCred, lbbg)
		if err != nil {
			return err
		}
	}
	return err
}

func (lbbg *SAwsCachedLbbg) SyncWithCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider) error {
	lbbg.SetModelManager(AwsCachedLbbgManager, lbbg)
	diff, err := db.UpdateWithLock(ctx, lbbg, func() error {
		lbbg.Status = extLoadbalancerBackendgroup.GetStatus()
		metadata := extLoadbalancerBackendgroup.GetMetadata()
		if port, _ := metadata.Int("port"); port > 0 {
			lbbg.Port = int(port)
		}
		if protocol, _ := metadata.GetString("health_check_protocol"); len(protocol) > 0 {
			lbbg.HealthCheckProtocol = protocol
		}
		if interval, _ := metadata.Int("health_check_interval"); interval > 0 {
			lbbg.HealthCheckInterval = int(interval)
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbbg, diff, userCred)

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, lb.ManagerId)
	return err
}

func (man *SAwsCachedLbbgManager) newFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider) (*SAwsCachedLbbg, error) {
	LocalLbbg, err := newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, extLoadbalancerBackendgroup, syncOwnerId)
	if err != nil {
		return nil, err
	}

	lbbg := &SAwsCachedLbbg{}
	lbbg.SetModelManager(man, lbbg)

	lbbg.ManagerId = lb.ManagerId
	lbbg.CloudregionId = lb.CloudregionId
	lbbg.LoadbalancerId = lb.Id
	lbbg.BackendGroupId = LocalLbbg.GetId()
	lbbg.ExternalId = extLoadbalancerBackendgroup.GetGlobalId()
	lbbg.ProtocolType = extLoadbalancerBackendgroup.GetProtocolType()

	metadata := extLoadbalancerBackendgroup.GetMetadata()
	if t, _ := metadata.GetString("target_type"); len(t) > 0 {
		lbbg.TargetType = t
	}

	if p, _ := metadata.Int("port"); p > 0 {
		lbbg.Port = int(p)
	}

	if protocol, _ := metadata.GetString("health_check_protocol"); len(protocol) > 0 {
		lbbg.HealthCheckProtocol = protocol
	}

	if interval, _ := metadata.Int("health_check_interval"); interval > 0 {
		lbbg.HealthCheckInterval = int(interval)
	}
	newName, err := db.GenerateName(man, syncOwnerId, LocalLbbg.GetName())
	if err != nil {
		return nil, err
	}

	lbbg.Name = newName
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

	err = man.TableSpec().Insert(lbbg)
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, lb.ManagerId)

	db.OpsLog.LogEvent(lbbg, db.ACT_CREATE, lbbg.GetShortDesc(ctx), userCred)
	return lbbg, nil
}
