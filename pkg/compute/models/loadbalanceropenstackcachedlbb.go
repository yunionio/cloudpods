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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SOpenstackCachedLbManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var OpenstackCachedLbManager *SOpenstackCachedLbManager

func init() {
	OpenstackCachedLbManager = &SOpenstackCachedLbManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SOpenstackCachedLb{},
			"openstackcachedlbbs_tbl",
			"openstackcachedlbb",
			"openstackcachedlbbs",
		),
	}
	OpenstackCachedLbManager.SetVirtualObject(OpenstackCachedLbManager)
}

type SOpenstackCachedLb struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	BackendServerId      string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // 后端服务器 实例ID
	BackendId            string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // 本地loadbalancebackend id
	CachedBackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (man *SOpenstackCachedLbManager) GetBackendsByLocalBackendId(backendId string) ([]SOpenstackCachedLb, error) {
	loadbalancerBackends := []SOpenstackCachedLb{}
	q := man.Query().IsFalse("pending_deleted").Equals("backend_id", backendId)
	if err := db.FetchModelObjects(man, q, &loadbalancerBackends); err != nil {
		return nil, err
	}
	return loadbalancerBackends, nil
}

func (man *SOpenstackCachedLbManager) CreateOpenstackCachedLb(ctx context.Context, userCred mcclient.TokenCredential, lbb *SLoadbalancerBackend, cachedLbbg *SOpenstackCachedLbbg, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider) (*SOpenstackCachedLb, error) {
	cachedlbb := &SOpenstackCachedLb{}
	cachedlbb.SetModelManager(man, cachedlbb)

	cachedlbb.CloudregionId = cachedLbbg.CloudregionId
	cachedlbb.ManagerId = cachedLbbg.ManagerId
	cachedlbb.CachedBackendGroupId = cachedLbbg.GetId()
	cachedlbb.BackendId = lbb.GetId()
	cachedlbb.ExternalId = extLoadbalancerBackend.GetGlobalId()

	if err := cachedlbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend); err != nil {
		return nil, err
	}

	var err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, syncOwnerId, extLoadbalancerBackend.GetName())
		if err != nil {
			return err
		}
		cachedlbb.Name = newName

		return man.TableSpec().Insert(ctx, cachedlbb)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, lbb, syncOwnerId, extLoadbalancerBackend, cachedLbbg.ManagerId)

	db.OpsLog.LogEvent(cachedlbb, db.ACT_CREATE, lbb.GetShortDesc(ctx), userCred)

	return cachedlbb, nil
}

func (lbb *SOpenstackCachedLb) GetCachedBackendGroup() (*SOpenstackCachedLbbg, error) {
	lbbg, err := db.FetchById(OpenstackCachedLbbgManager, lbb.CachedBackendGroupId)
	if err != nil {
		return nil, err
	}

	return lbbg.(*SOpenstackCachedLbbg), nil
}

func (man *SOpenstackCachedLbManager) getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup *SOpenstackCachedLbbg) ([]SOpenstackCachedLb, error) {
	loadbalancerBackends := []SOpenstackCachedLb{}
	q := man.Query().Equals("cached_backend_group_id", loadbalancerBackendgroup.Id)
	if err := db.FetchModelObjects(man, q, &loadbalancerBackends); err != nil {
		return nil, err
	}
	return loadbalancerBackends, nil
}

func (man *SOpenstackCachedLbManager) SyncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, loadbalancerBackendgroup *SOpenstackCachedLbbg, lbbs []cloudprovider.ICloudLoadbalancerBackend, syncRange *SSyncRange) compare.SyncResult {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "backends", loadbalancerBackendgroup.Id)
	defer lockman.ReleaseRawObject(ctx, "backends", loadbalancerBackendgroup.Id)

	syncResult := compare.SyncResult{}

	dbLbbs, err := man.getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SOpenstackCachedLb{}
	commondb := []SOpenstackCachedLb{}
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
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerBackend(ctx, userCred, loadbalancerBackendgroup, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (lbb *SOpenstackCachedLb) syncRemoveCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		lbb.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
		return errors.Wrap(err, "lbb.ValidateDeleteCondition(ctx)")
	}
	lbb.SetModelManager(OpenstackCachedLbManager, lbb)
	err = db.DeleteModel(ctx, userCred, lbb)
	if err != nil {
		return err
	}
	return nil
}

func (lbb *SOpenstackCachedLb) constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend) error {
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
	return nil
}

func (lbb *SOpenstackCachedLb) SyncWithCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider) error {
	lbb.SetModelManager(OpenstackCachedLbManager, lbb)
	cacheLbbg, err := lbb.GetCachedBackendGroup()
	if err != nil {
		return errors.Wrap(err, "OpenstackCachedLb.SyncWithCloudLoadbalancerBackend.GetCachedBackendGroup")
	}

	localLbbg, err := cacheLbbg.GetLocalBackendGroup(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "OpenstackCachedLb.SyncWithCloudLoadbalancerBackend.GetLocalBackendGroup")
	}

	locallbb, err := newLocalBackendFromCloudLoadbalancerBackend(ctx, userCred, localLbbg, extLoadbalancerBackend, syncOwnerId)
	if err != nil {
		return errors.Wrap(err, "OpenstackCachedLb.SyncWithCloudLoadbalancerBackend.newLocalBackendFromCloudLoadbalancerBackend")
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

func (man *SOpenstackCachedLbManager) newFromCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, loadbalancerBackendgroup *SOpenstackCachedLbbg, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider) (*SOpenstackCachedLb, error) {
	localBackendGroup, err := loadbalancerBackendgroup.GetLocalBackendGroup(ctx, userCred)
	if err != nil {
		return nil, err
	}
	// openstack lb后端是ip:port 不一定有server映射
	if len(extLoadbalancerBackend.GetBackendId()) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "extLoadbalancerBackend.GetBackendId()")
	}
	locallbb, err := newLocalBackendFromCloudLoadbalancerBackend(ctx, userCred, localBackendGroup, extLoadbalancerBackend, syncOwnerId)
	if err != nil {
		return nil, err
	}
	lbb := &SOpenstackCachedLb{}
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
