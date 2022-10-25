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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SCachedLoadbalancerAclManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SLoadbalancerAclResourceBaseManager
}

var CachedLoadbalancerAclManager *SCachedLoadbalancerAclManager

func init() {
	CachedLoadbalancerAclManager = &SCachedLoadbalancerAclManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SCachedLoadbalancerAcl{},
			"cachedloadbalanceracls_tbl",
			"cachedloadbalanceracl",
			"cachedloadbalanceracls",
		),
	}

	CachedLoadbalancerAclManager.SetVirtualObject(CachedLoadbalancerAclManager)
}

type SCachedLoadbalancerAcl struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase
	SLoadbalancerAclResourceBase

	ListenerId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // huawei only
}

func (manager *SCachedLoadbalancerAclManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (lbacl *SCachedLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, true)
	if err != nil {
		return nil, err
	}
	input := apis.VirtualResourceBaseUpdateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lbacl.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (lbacl *SCachedLoadbalancerAcl) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	lbacl.SetStatus(userCred, api.LB_SYNC_CONF, "")
	lbacl.StartLoadBalancerAclSyncTask(ctx, userCred, "")
}

func (lbacl *SCachedLoadbalancerAcl) StartLoadBalancerAclSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclSyncTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (man *SCachedLoadbalancerAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	aclV := validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", ownerId)
	regionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", ownerId)
	providerV := validators.NewModelIdOrNameValidator("cloudprovider", "cloudprovider", ownerId)
	keyV := map[string]validators.IValidator{
		"acl":           aclV,
		"cloudregion":   regionV,
		"cloudprovider": providerV,
	}

	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}

	if utils.IsInStringArray(providerV.Model.(*SCloudprovider).Provider, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
		listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", ownerId)
		if err := listenerV.Validate(data); err != nil {
			return nil, err
		}
	} else {
		data.Remove("listener_id")
	}

	q := man.Query().Equals("acl_id", aclV.Model.GetId()).Equals("cloudregion_id", regionV.Model.GetId()).IsFalse("deleted")
	if listener, _ := data.GetString("listener_id"); len(listener) > 0 {
		q.Equals("listener_id", listener)
	}

	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, httperrors.NewDuplicateResourceError("the acl cache in region %s aready exists.", regionV.Model.GetId())
	}

	provider := providerV.Model.(*SCloudprovider)
	data.Set("manager_id", jsonutils.NewString(provider.Id))
	name, _ := db.GenerateName(ctx, man, ownerId, aclV.Model.GetName())
	data.Set("name", jsonutils.NewString(name))

	input := apis.VirtualResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	return data, nil
}

func (lbacl *SCachedLoadbalancerAcl) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)

	lbacl.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbacl.StartLoadBalancerAclCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalanceracl error: %v", err)
	}
}

func (lbacl *SCachedLoadbalancerAcl) StartLoadBalancerAclCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclCreateTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbacl *SCachedLoadbalancerAcl) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(lbacl.CloudregionId)
	if err != nil {
		log.Errorf("failed to find region for loadbalancer acl %s", lbacl.Name)
		return nil
	}
	return region.(*SCloudregion)
}

func (lbacl *SCachedLoadbalancerAcl) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := lbacl.GetDriver(ctx)
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for lb %s: %s", lbacl.Name, err)
	}
	region := lbacl.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find region for lb %s", lbacl.Name)
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (lbacl *SCachedLoadbalancerAcl) GetListener() (*SLoadbalancerListener, error) {
	if len(lbacl.ListenerId) == 0 {
		return nil, fmt.Errorf("acl %s has no listener", lbacl.Id)
	}

	listener, err := LoadbalancerListenerManager.FetchById(lbacl.ListenerId)
	if err != nil {
		return nil, err
	}

	return listener.(*SLoadbalancerListener), nil
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

	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := man.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := man.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.CachedLoadbalancerAclDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regionRows[i],
		}
	}

	return rows
}

func (lbacl *SCachedLoadbalancerAcl) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	man := LoadbalancerListenerManager
	t := man.TableSpec().Instance()
	pdF := t.Field("pending_deleted")
	lbaclId := lbacl.Id
	n, err := t.Query().
		Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
		Equals("domain_id", lbacl.DomainId).
		Equals("acl_id", lbaclId).
		Equals("cached_acl_id", lbaclId).
		CountWithError()
	if err != nil {
		return httperrors.NewInternalServerError("get acl count fail %s", err)
	}
	if n > 0 {
		// return fmt.Errorf("acl %s is still referred to by %d %s",
		// 	lbaclId, n, man.KeywordPlural())
		return httperrors.NewResourceBusyError("acl %s is still referred to by %d %s", lbaclId, n, man.KeywordPlural())
	}
	return nil
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
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbacl *SCachedLoadbalancerAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SCachedLoadbalancerAcl) syncRemoveCloudLoadbalanceAcl(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		err = self.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		self.DoPendingDelete(ctx, userCred)
	}
	return errors.Wrap(err, "cachedLoadbalancerAcl.remove.Delete")
}

func (acl *SCachedLoadbalancerAcl) SyncWithCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, extAcl cloudprovider.ICloudLoadbalancerAcl, projectId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, acl, func() error {
		// todo: 华为云acl没有name字段应此不需要同步名称
		if !utils.IsInStringArray(acl.GetProviderName(), []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
			acl.Name = extAcl.GetName()
		} else {
			ext_listener_id := extAcl.GetAclListenerID()
			if len(ext_listener_id) > 0 {
				ilistener, err := db.FetchByExternalIdAndManagerId(LoadbalancerListenerManager, ext_listener_id, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
					sq := LoadbalancerManager.Query().SubQuery()
					return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("loadbalancer_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), acl.ManagerId))
				})
				if err != nil {
					return errors.Wrap(err, "cacheLoadbalancerAcl.sync.FetchByExternalId")
				}

				acl.ListenerId = ilistener.(*SLoadbalancerListener).GetId()
			}
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "cacheLoadbalancerAcl.sync.Update")
	}
	db.OpsLog.LogSyncUpdate(acl, diff, userCred)

	SyncCloudProject(userCred, acl, projectId, extAcl, acl.ManagerId)

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
	lbacl.ProjectId = lblis.ProjectId
	lbacl.ProjectSrc = lblis.ProjectSrc
	lbacl.Name = acl.Name
	lbacl.AclId = acl.Id
	lbacl.ListenerId = listenerId

	err = man.TableSpec().Insert(ctx, &lbacl)
	if err != nil {
		return nil, err
	}

	return &lbacl, err
}

func (man *SCachedLoadbalancerAclManager) getLoadbalancerAclsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SCachedLoadbalancerAcl, error) {
	acls := []SCachedLoadbalancerAcl{}
	q := man.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &acls); err != nil {
		log.Errorf("failed to get acls for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return acls, nil
}

func (man *SCachedLoadbalancerAclManager) getLoadbalancerAclByRegion(provider *SCloudprovider, regionId string, aclId string, listenerId string) (SCachedLoadbalancerAcl, error) {
	acls := []SCachedLoadbalancerAcl{}
	q := man.Query().Equals("cloudregion_id", regionId).Equals("manager_id", provider.Id).IsFalse("pending_deleted")
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

	aclEntites := SLoadbalancerAclEntries{}
	for _, entry := range extAcl.GetAclEntries() {
		aclEntites = append(aclEntites, &SLoadbalancerAclEntry{Cidr: entry.CIDR, Comment: entry.Comment})
	}

	f := aclEntites.Fingerprint()
	if LoadbalancerAclManager.CountByFingerPrint(provider.ProjectId, f) == 0 {
		localAcl := &SLoadbalancerAcl{}
		localAcl.SetModelManager(LoadbalancerAclManager, localAcl)
		localAcl.Name = acl.Name
		localAcl.Description = acl.Description
		localAcl.AclEntries = &aclEntites
		localAcl.Fingerprint = f
		localAcl.IsPublic = true
		localAcl.PublicScope = string(rbacutils.ScopeDomain)
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

	SyncCloudProject(userCred, &acl, projectId, extAcl, acl.ManagerId)

	db.OpsLog.LogEvent(&acl, db.ACT_CREATE, acl.GetShortDesc(ctx), userCred)

	return &acl, nil
}

func (manager *SCachedLoadbalancerAclManager) InitializeData() error {
	// todo: sync old data from acls
	return nil
}

func (manager *SCachedLoadbalancerAclManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CachedLoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
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

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
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

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
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
