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
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
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
	q := man.Query().IsFalse("pending_deleted").Equals("backend_group_id", backendGroupId)
	err := db.FetchModelObjects(man, q, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (man *SAwsCachedLbbgManager) getLoadbalancerBackendgroupsByRegion(managerId string, regionId string) ([]SAwsCachedLbbg, error) {
	lbbgs := []SAwsCachedLbbg{}
	q := man.Query().Equals("cloudregion_id", regionId).Equals("manager_id", managerId).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for region: %s error: %v", regionId, err)
		return nil, err
	}
	return lbbgs, nil
}

func (man *SAwsCachedLbbgManager) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, lbbgs []cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) ([]SAwsCachedLbbg, []cloudprovider.ICloudLoadbalancerBackendGroup, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "backendgroups", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "backendgroups", fmt.Sprintf("%s-%s", provider.Id, region.Id))

	localLbgs := []SAwsCachedLbbg{}
	remoteLbbgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	syncResult := compare.SyncResult{}

	dbLbbgs, err := man.getLoadbalancerBackendgroupsByRegion(provider.GetId(), region.GetId())
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
		elbId := commonext[i].GetLoadbalancerId()
		if len(elbId) > 0 {
			ielb, err := db.FetchByExternalIdAndManagerId(LoadbalancerManager, elbId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", provider.Id)
			})
			if err == nil {
				elb = ielb.(*SLoadbalancer)
			}
		}

		if elb == nil {
			log.Debugf("Aws.SyncLoadbalancerBackendgroups skiped external backendgroup %s", elbId)
			continue
		}

		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, elb, commonext[i], provider.GetOwnerId(), provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localLbgs = append(localLbgs, commondb[i])
			remoteLbbgs = append(remoteLbbgs, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		var elb *SLoadbalancer
		elbId := added[i].GetLoadbalancerId()
		if len(elbId) > 0 {
			elb, err = LoadbalancerManager.FetchByExternalId(provider.GetId(), elbId)
			if err != nil {
				log.Debugf("awsCachedLbbgManager.SyncLoadbalancerBackendgroups %s", err)
				syncResult.AddError(err)
				continue
			}
		}

		if elb == nil {
			log.Debugf("Aws.SyncLoadbalancerBackendgroups skiped external backendgroup %s", elbId)
			continue
		}

		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, elb, added[i], syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
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

func (lbbg *SAwsCachedLbbg) isBackendsMatch(backends []SLoadbalancerBackend, ibackends []cloudprovider.ICloudLoadbalancerBackend) bool {
	if len(ibackends) != len(backends) {
		return false
	}

	locals := []string{}
	remotes := []string{}

	for i := range backends {
		guest := backends[i].GetGuest()
		seg := strings.Join([]string{guest.ExternalId, strconv.Itoa(backends[i].Port)}, "/")
		locals = append(locals, seg)
	}

	for i := range ibackends {
		ibackend := ibackends[i]
		seg := strings.Join([]string{ibackend.GetBackendId(), strconv.Itoa(ibackend.GetPort())}, "/")
		remotes = append(remotes, seg)
	}

	for i := range remotes {
		if !utils.IsInStringArray(remotes[i], locals) {
			return false
		}
	}

	return true
}

func (lbbg *SAwsCachedLbbg) SyncWithCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) error {
	lbbg.SetModelManager(AwsCachedLbbgManager, lbbg)

	ibackends, err := extLoadbalancerBackendgroup.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "AwsCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.GetILoadbalancerBackends")
	}

	localLbbg, err := lbbg.GetLocalBackendGroup(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "AwsCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.GetLocalBackendGroup")
	}

	backends, err := localLbbg.GetBackends()
	if err != nil {
		return errors.Wrap(err, "AwsCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.GetBackends")
	}

	var newLocalLbbg *SLoadbalancerBackendGroup
	if !lbbg.isBackendsMatch(backends, ibackends) {
		newLocalLbbg, err = newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, extLoadbalancerBackendgroup, syncOwnerId, provider)
		if err != nil {
			return errors.Wrap(err, "HuaweiCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.newLocalBackendgroupFromCloudLoadbalancerBackendgroup")
		}
	}

	diff, err := db.UpdateWithLock(ctx, lbbg, func() error {
		lbbg.Status = extLoadbalancerBackendgroup.GetStatus()
		metadata := extLoadbalancerBackendgroup.GetSysTags()
		if port, ok := metadata["port"]; ok {
			portNum, err := strconv.Atoi(port)
			if err == nil {
				lbbg.Port = portNum
			}
		}
		if protocol, ok := metadata["health_check_protocol"]; ok {
			lbbg.HealthCheckProtocol = protocol
		}
		if interval, ok := metadata["health_check_interval"]; ok {
			intervalNum, err := strconv.Atoi(interval)
			if err == nil {
				lbbg.HealthCheckInterval = intervalNum
			}
		}
		if newLocalLbbg != nil {
			lbbg.BackendGroupId = newLocalLbbg.GetId()
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbbg, diff, userCred)

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, provider.Id)
	return err
}

func (man *SAwsCachedLbbgManager) newFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) (*SAwsCachedLbbg, error) {
	LocalLbbg, err := newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, extLoadbalancerBackendgroup, syncOwnerId, provider)
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

	metadata := extLoadbalancerBackendgroup.GetSysTags()
	if t, ok := metadata["target_type"]; ok {
		lbbg.TargetType = t
	}

	if p, ok := metadata["port"]; ok {
		portNum, err := strconv.Atoi(p)
		if err == nil {
			lbbg.Port = portNum
		}
	}

	if protocol, ok := metadata["health_check_protocol"]; ok {
		lbbg.HealthCheckProtocol = protocol
	}

	if interval, ok := metadata["health_check_interval"]; ok {
		intervalNum, err := strconv.Atoi(interval)
		if err == nil {
			lbbg.HealthCheckInterval = intervalNum
		}
	}
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

	err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		lbbg.Name, err = db.GenerateName(ctx, man, syncOwnerId, LocalLbbg.GetName())
		if err != nil {
			return err
		}

		return man.TableSpec().Insert(ctx, lbbg)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, provider.Id)

	db.OpsLog.LogEvent(lbbg, db.ACT_CREATE, lbbg.GetShortDesc(ctx), userCred)
	return lbbg, nil
}

func (man *SAwsCachedLbbgManager) InitializeData() error {
	ret := []db.SMetadata{}
	q := db.Metadata.Query().Equals("obj_type", man.Keyword())
	err := db.FetchModelObjects(db.Metadata, q, &ret)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Debugf("SAwsCachedLbbgManager.InitializeData %s", err)
		}

		return nil
	}

	for i := range ret {
		item := ret[i]
		_, err := db.Update(&item, func() error {
			return item.MarkDelete()
		})
		if err != nil {
			log.Debugf("SAwsCachedLbbgManager.MarkDelete %s", err)
			return nil
		}
	}

	log.Debugf("SAwsCachedLbbgManager cleaned %d dirty data.", len(ret))
	return nil
}
