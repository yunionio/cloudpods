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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SCachedLoadbalancerAclManager struct {
	SLoadbalancerLogSkipper
	db.SStatusStandaloneResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SLoadbalancerAclResourceBaseManager
}

var CachedLoadbalancerAclManager *SCachedLoadbalancerAclManager

func init() {
	CachedLoadbalancerAclManager = &SCachedLoadbalancerAclManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SCachedLoadbalancerAcl{},
			"cachedloadbalanceracls_tbl",
			"cachedloadbalanceracl",
			"cachedloadbalanceracls",
		),
	}

	CachedLoadbalancerAclManager.SetVirtualObject(CachedLoadbalancerAclManager)
}

type SCachedLoadbalancerAcl struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase
	SLoadbalancerAclResourceBase
}

func (manager *SCachedLoadbalancerAclManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (self *SCachedLoadbalancerAcl) GetOwnerId() mcclient.IIdentityProvider {
	acl, err := self.GetAcl()
	if err != nil {
		return nil
	}
	return acl.GetOwnerId()
}

func (manager *SCachedLoadbalancerAclManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	aclId, _ := data.GetString("acl_id")
	if len(aclId) > 0 {
		cert, err := db.FetchById(LoadbalancerAclManager, aclId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(LoadbalancerAclManager, %s)", aclId)
		}
		return cert.(*SLoadbalancerAcl).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (manager *SCachedLoadbalancerAclManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := LoadbalancerAclManager.Query("id")
		switch scope {
		case rbacscope.ScopeProject:
			sq = sq.Equals("tenant_id", userCred.GetProjectId())
			return q.In("acl_id", sq.SubQuery())
		case rbacscope.ScopeDomain:
			sq = sq.Equals("domain_id", userCred.GetProjectDomainId())
			return q.In("acl_id", sq.SubQuery())
		}
	}
	return q
}

func (lbacl *SCachedLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.CachedLoadbalancerAclUpdateInput) (*api.CachedLoadbalancerAclUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = lbacl.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (lbacl *SCachedLoadbalancerAcl) StartLoadBalancerAclSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclSyncTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (man *SCachedLoadbalancerAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (lbacl *SCachedLoadbalancerAcl) StartLoadBalancerAclCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclCreateTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lbacl *SCachedLoadbalancerAcl) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(lbacl.CloudregionId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion.FetchById(%s)", lbacl.CloudregionId)
	}
	return region.(*SCloudregion), nil
}

func (lbacl *SCachedLoadbalancerAcl) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := lbacl.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	region, err := lbacl.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (man *SCachedLoadbalancerAclManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CachedLoadbalancerAclDetails {
	rows := make([]api.CachedLoadbalancerAclDetails, len(objs))

	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := man.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := man.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.CachedLoadbalancerAclDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ManagedResourceInfo:             manRows[i],
			CloudregionResourceInfo:         regionRows[i],
		}
	}

	return rows
}

func (lbacl *SCachedLoadbalancerAcl) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "purge")
	return nil, lbacl.StartLoadBalancerAclDeleteTask(ctx, userCred, params, "")
}

func (lbacl *SCachedLoadbalancerAcl) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbacl.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lbacl.StartLoadBalancerAclDeleteTask(ctx, userCred, query.(*jsonutils.JSONDict), "")
}

func (lbacl *SCachedLoadbalancerAcl) StartLoadBalancerAclDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclDeleteTask", lbacl, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lbacl *SCachedLoadbalancerAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lbacl *SCachedLoadbalancerAcl) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return lbacl.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SCachedLoadbalancerAcl) syncRemoveCloudLoadbalanceAcl(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.RealDelete(ctx, userCred)
}

func (acl *SCachedLoadbalancerAcl) SyncWithCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, extAcl cloudprovider.ICloudLoadbalancerAcl, projectId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, acl, func() error {
		// todo: 华为云acl没有name字段应此不需要同步名称
		if !utils.IsInStringArray(acl.GetProviderName(), []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
			acl.Name = extAcl.GetName()
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "cacheLoadbalancerAcl.sync.Update")
	}
	db.OpsLog.LogSyncUpdate(acl, diff, userCred)
	return nil
}

func (man *SCachedLoadbalancerAclManager) GetOrCreateCachedAcl(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lblis *SLoadbalancerListener, acl *SLoadbalancerAcl) (*SCachedLoadbalancerAcl, error) {
	ownerProjId := provider.ProjectId

	lockman.LockClass(ctx, man, ownerProjId)
	defer lockman.ReleaseClass(ctx, man, ownerProjId)

	listenerId := ""
	if utils.IsInStringArray(lblis.GetProviderName(), []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
		listenerId = lblis.Id
	}
	if lblis.GetProviderName() == api.CLOUD_PROVIDER_OPENSTACK {
		listenerId = lblis.Id
	}

	region, err := lblis.GetRegion()
	if err != nil {
		return nil, err
	}
	lbacl, err := man.getLoadbalancerAclByRegion(provider, region.Id, acl.Id, listenerId)
	if err == nil {
		if lbacl.Id != acl.Id {
			_, err := man.TableSpec().Update(ctx, &lbacl, func() error {
				lbacl.Name = acl.Name
				lbacl.AclId = acl.Id
				return nil
			})

			if err != nil {
				return nil, err
			}
		}
		return &lbacl, nil
	}

	if err.Error() != "NotFound" {
		return nil, err
	}

	lbacl = SCachedLoadbalancerAcl{}
	lbacl.ManagerId = provider.Id
	lbacl.CloudregionId = region.Id
	lbacl.Name = acl.Name
	lbacl.AclId = acl.Id

	err = man.TableSpec().Insert(ctx, &lbacl)
	if err != nil {
		return nil, err
	}

	return &lbacl, err
}

func (man *SCachedLoadbalancerAclManager) getLoadbalancerAclsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SCachedLoadbalancerAcl, error) {
	acls := []SCachedLoadbalancerAcl{}
	q := man.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id)
	if err := db.FetchModelObjects(man, q, &acls); err != nil {
		log.Errorf("failed to get acls for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return acls, nil
}

func (man *SCachedLoadbalancerAclManager) getLoadbalancerAclByRegion(provider *SCloudprovider, regionId string, aclId string, listenerId string) (SCachedLoadbalancerAcl, error) {
	acls := []SCachedLoadbalancerAcl{}
	q := man.Query().Equals("cloudregion_id", regionId).Equals("manager_id", provider.Id)
	// used by huawei only
	if len(listenerId) > 0 {
		q.Equals("listener_id", listenerId)
	} else {
		q.Equals("acl_id", aclId)
	}

	if err := db.FetchModelObjects(man, q, &acls); err != nil {
		log.Errorf("failed to get acl for region: %v provider: %v error: %v", regionId, provider, err)
		return SCachedLoadbalancerAcl{}, err
	}

	if len(acls) == 1 {
		return acls[0], nil
	} else if len(acls) == 0 {
		return SCachedLoadbalancerAcl{}, fmt.Errorf("NotFound")
	} else {
		return SCachedLoadbalancerAcl{}, fmt.Errorf("Duplicate acl %s found for region %s", aclId, regionId)
	}
}

func (man *SCachedLoadbalancerAclManager) SyncLoadbalancerAcls(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, acls []cloudprovider.ICloudLoadbalancerAcl, syncRange *SSyncRange) compare.SyncResult {
	lockman.LockRawObject(ctx, "acls", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "acls", fmt.Sprintf("%s-%s", provider.Id, region.Id))

	syncResult := compare.SyncResult{}

	dbAcls, err := man.getLoadbalancerAclsByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SCachedLoadbalancerAcl{}
	commondb := []SCachedLoadbalancerAcl{}
	commonext := []cloudprovider.ICloudLoadbalancerAcl{}
	added := []cloudprovider.ICloudLoadbalancerAcl{}

	err = compare.CompareSets(dbAcls, acls, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalanceAcl(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerAcl(ctx, userCred, commonext[i], provider.GetOwnerId())
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerAcl(ctx, userCred, provider, added[i], region, provider.GetOwnerId())
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (man *SCachedLoadbalancerAclManager) newFromCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extAcl cloudprovider.ICloudLoadbalancerAcl, region *SCloudregion, projectId mcclient.IIdentityProvider) (*SCachedLoadbalancerAcl, error) {
	acl := SCachedLoadbalancerAcl{}
	acl.SetModelManager(man, &acl)

	acl.ExternalId = extAcl.GetGlobalId()
	acl.ManagerId = provider.Id
	acl.CloudregionId = region.Id

	aclEntites := api.SAclEntries{}
	for _, entry := range extAcl.GetAclEntries() {
		aclEntites = append(aclEntites, api.SAclEntry{Cidr: entry.CIDR, Comment: entry.Comment})
	}

	f := aclEntites.GetFingerprint()
	if LoadbalancerAclManager.CountByFingerPrint(provider.ProjectId, f) == 0 {
		localAcl := &SLoadbalancerAcl{}
		localAcl.SetModelManager(LoadbalancerAclManager, localAcl)
		localAcl.Name = acl.Name
		localAcl.Description = acl.Description
		localAcl.AclEntries = &aclEntites
		localAcl.Fingerprint = f
		localAcl.IsPublic = true
		localAcl.PublicScope = string(rbacscope.ScopeDomain)
		err := LoadbalancerAclManager.TableSpec().Insert(ctx, localAcl)
		if err != nil {
			return nil, errors.Wrap(err, "cachedLoadbalancerAclManager.new.InsertAcl")
		}

		SyncCloudProject(userCred, localAcl, provider.GetOwnerId(), extAcl, provider.GetId())
	}

	{
		localAcl, err := LoadbalancerAclManager.FetchByFingerPrint(provider.ProjectId, f)
		if err != nil {
			return nil, errors.Wrap(err, "cachedLoadbalancerAclManager.new.FetchByFingerPrint")
		}

		acl.AclId = localAcl.GetId()
	}

	var err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, projectId, extAcl.GetName())
		if err != nil {
			return errors.Wrap(err, "cachedLoadbalancerAclManager.new.GenerateName")
		}
		acl.Name = newName
		return man.TableSpec().Insert(ctx, &acl)
	}()
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	db.OpsLog.LogEvent(&acl, db.ACT_CREATE, acl.GetShortDesc(ctx), userCred)
	return &acl, nil
}

func (manager *SCachedLoadbalancerAclManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CachedLoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SLoadbalancerAclResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerAclFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerAclResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SCachedLoadbalancerAclManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CachedLoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SLoadbalancerAclResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerAclFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerAclResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SCachedLoadbalancerAclManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SLoadbalancerAclResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SCachedLoadbalancerAclManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SLoadbalancerAclResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerAclResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerAclResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SCachedLoadbalancerAclManager) InitializeData() error {
	return nil
}
