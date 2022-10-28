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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SQcloudCachedLbManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var QcloudCachedLbManager *SQcloudCachedLbManager

func init() {
	QcloudCachedLbManager = &SQcloudCachedLbManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SQcloudCachedLb{},
			"qcloudcachedlbbs_tbl",
			"qcloudcachedlbb",
			"qcloudcachedlbbs",
		),
	}
	QcloudCachedLbManager.SetVirtualObject(QcloudCachedLbManager)
}

type SQcloudCachedLb struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	BackendServerId      string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // 后端服务器 实例ID
	BackendId            string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // 本地loadbalancebackend id
	CachedBackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (manager *SQcloudCachedLbManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (lbb *SQcloudCachedLb) GetCustomizeColumns(context.Context, mcclient.TokenCredential, jsonutils.JSONObject) *jsonutils.JSONDict {
	return nil
}

func (lbb *SQcloudCachedLb) GetCachedBackendGroup() (*SQcloudCachedLbbg, error) {
	lbbg, err := db.FetchById(QcloudCachedLbbgManager, lbb.CachedBackendGroupId)
	if err != nil {
		return nil, err
	}

	return lbbg.(*SQcloudCachedLbbg), nil
}

func (lbb *SQcloudCachedLb) syncRemoveCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		err = lbb.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		lbb.SetModelManager(QcloudCachedLbManager, lbb)
		err := db.DeleteModel(ctx, userCred, lbb)
		if err != nil {
			return err
		}
	}
	return err
}

func (lbb *SQcloudCachedLb) constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend) error {
	lbb.Status = extLoadbalancerBackend.GetStatus()

	instance, err := db.FetchByExternalIdAndManagerId(GuestManager, extLoadbalancerBackend.GetBackendId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := HostManager.Query().SubQuery()
		return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("host_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), lbb.ManagerId))
	})
	if err != nil {
		return err
	}
	guest := instance.(*SGuest)

	lbb.BackendServerId = guest.Id
	lbb.ExternalId = extLoadbalancerBackend.GetGlobalId()
	return nil
}

func (lbb *SQcloudCachedLb) SyncWithCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider) error {
	lbb.SetModelManager(QcloudCachedLbManager, lbb)
	cacheLbbg, err := lbb.GetCachedBackendGroup()
	if err != nil {
		return errors.Wrap(err, "QcloudCachedLb.SyncWithCloudLoadbalancerBackend.GetCachedBackendGroup")
	}

	localLbbg, err := cacheLbbg.GetLocalBackendGroup(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "QcloudCachedLb.SyncWithCloudLoadbalancerBackend.GetLocalBackendGroup")
	}

	locallbb, err := newLocalBackendFromCloudLoadbalancerBackend(ctx, userCred, localLbbg, extLoadbalancerBackend, syncOwnerId)
	if err != nil {
		return errors.Wrap(err, "QcloudCachedLb.SyncWithCloudLoadbalancerBackend.newLocalBackendFromCloudLoadbalancerBackend")
	}

	diff, err := db.UpdateWithLock(ctx, lbb, func() error {
		if locallbb != nil {
			lbb.BackendId = locallbb.GetId()
		}

		return lbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend)
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbb, diff, userCred)

	SyncCloudProject(userCred, lbb, syncOwnerId, extLoadbalancerBackend, lbb.ManagerId)

	return nil
}

func (man *SQcloudCachedLbManager) newFromCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, loadbalancerBackendgroup *SQcloudCachedLbbg, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider) (*SQcloudCachedLb, error) {
	localBackendGroup, err := loadbalancerBackendgroup.GetLocalBackendGroup(ctx, userCred)
	if err != nil {
		return nil, err
	} else if localBackendGroup == nil {
		return nil, fmt.Errorf("newFromCloudLoadbalancerBackend localBackendGroup is nil")
	}

	locallbb, err := newLocalBackendFromCloudLoadbalancerBackend(ctx, userCred, localBackendGroup, extLoadbalancerBackend, syncOwnerId)
	if err != nil {
		return nil, err
	}
	lbb := &SQcloudCachedLb{}
	lbb.SetModelManager(man, lbb)

	lbb.CloudregionId = loadbalancerBackendgroup.CloudregionId
	lbb.ManagerId = loadbalancerBackendgroup.ManagerId
	lbb.CachedBackendGroupId = loadbalancerBackendgroup.Id
	lbb.BackendId = locallbb.GetId()
	lbb.ExternalId = extLoadbalancerBackend.GetGlobalId()

	if err := lbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend); err != nil {
		return nil, err
	}

	err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		lbb.Name, err = db.GenerateName(ctx, man, syncOwnerId, extLoadbalancerBackend.GetName())
		if err != nil {
			return err
		}

		return man.TableSpec().Insert(ctx, lbb)
	}()
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, lbb, syncOwnerId, extLoadbalancerBackend, loadbalancerBackendgroup.ManagerId)

	db.OpsLog.LogEvent(lbb, db.ACT_CREATE, lbb.GetShortDesc(ctx), userCred)

	return lbb, nil
}

func (man *SQcloudCachedLbManager) getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup *SQcloudCachedLbbg) ([]SQcloudCachedLb, error) {
	loadbalancerBackends := []SQcloudCachedLb{}
	q := man.Query().Equals("cached_backend_group_id", loadbalancerBackendgroup.Id)
	if err := db.FetchModelObjects(man, q, &loadbalancerBackends); err != nil {
		return nil, err
	}
	return loadbalancerBackends, nil
}

func (man *SQcloudCachedLbManager) SyncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, loadbalancerBackendgroup *SQcloudCachedLbbg, lbbs []cloudprovider.ICloudLoadbalancerBackend, syncRange *SSyncRange) compare.SyncResult {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "backends", loadbalancerBackendgroup.Id)
	defer lockman.ReleaseRawObject(ctx, "backends", loadbalancerBackendgroup.Id)

	syncResult := compare.SyncResult{}

	dbLbbs, err := man.getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SQcloudCachedLb{}
	commondb := []SQcloudCachedLb{}
	commonext := []cloudprovider.ICloudLoadbalancerBackend{}
	added := []cloudprovider.ICloudLoadbalancerBackend{}

	err = compare.CompareSets(dbLbbs, lbbs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerBackend(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackend(ctx, userCred, commonext[i], syncOwnerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		_, err := man.newFromCloudLoadbalancerBackend(ctx, userCred, loadbalancerBackendgroup, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (man *SQcloudCachedLbManager) GetBackendsByLocalBackendId(backendId string) ([]SQcloudCachedLb, error) {
	loadbalancerBackends := []SQcloudCachedLb{}
	q := man.Query().IsFalse("pending_deleted").Equals("backend_id", backendId)
	if err := db.FetchModelObjects(man, q, &loadbalancerBackends); err != nil {
		return nil, err
	}
	return loadbalancerBackends, nil
}

func (man *SQcloudCachedLbManager) CreateQcloudCachedLb(ctx context.Context, userCred mcclient.TokenCredential, lbb *SLoadbalancerBackend, cachedLbbg *SQcloudCachedLbbg, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider) (*SQcloudCachedLb, error) {
	cachedlbb := &SQcloudCachedLb{}
	cachedlbb.SetModelManager(man, cachedlbb)

	cachedlbb.CloudregionId = cachedLbbg.CloudregionId
	cachedlbb.ManagerId = cachedLbbg.ManagerId
	cachedlbb.CachedBackendGroupId = cachedLbbg.GetId()
	cachedlbb.BackendId = lbb.GetId()
	cachedlbb.ExternalId = extLoadbalancerBackend.GetGlobalId()

	if err := cachedlbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend); err != nil {
		return nil, err
	}

	var err error
	err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		cachedlbb.Name, err = db.GenerateName(ctx, man, syncOwnerId, extLoadbalancerBackend.GetName())
		if err != nil {
			return err
		}

		return man.TableSpec().Insert(ctx, cachedlbb)
	}()
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, lbb, syncOwnerId, extLoadbalancerBackend, cachedLbbg.ManagerId)

	db.OpsLog.LogEvent(cachedlbb, db.ACT_CREATE, lbb.GetShortDesc(ctx), userCred)

	return cachedlbb, nil
}
