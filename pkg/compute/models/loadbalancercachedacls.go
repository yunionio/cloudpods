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
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCachedLoadbalancerAclManager struct {
	SLoadbalancerLogSkipper
	db.SSharableVirtualResourceBaseManager
}

var CachedLoadbalancerAclManager *SCachedLoadbalancerAclManager

func init() {
	CachedLoadbalancerAclManager = &SCachedLoadbalancerAclManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SCachedLoadbalancerAcl{},
			"cachedloadbalanceracls_tbl",
			"cachedloadbalanceracl",
			"cachedloadbalanceracls",
		),
	}

	CachedLoadbalancerAclManager.SetVirtualObject(CachedLoadbalancerAclManager)
}

type SCachedLoadbalancerAcl struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	AclId      string                   `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"` // 本地ACL ID
	AclEntries *SLoadbalancerAclEntries `list:"user" update:"user" create:"required"`
	ListenerId string                   `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"` // huawei only
}

func (man *SCachedLoadbalancerAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, false)
	if err != nil {
		return nil, err
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}

	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerProjId)
	managerIdV.Optional(true)
	if err := managerIdV.Validate(data); err != nil {
		return nil, err
	}

	regionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", ownerProjId)
	regionV.Default("default")
	if err := regionV.Validate(data); err != nil {
		return nil, err
	}
	region := regionV.Model.(*SCloudregion)
	return region.GetDriver().ValidateCreateLoadbalancerAclData(ctx, userCred, data)
}

func (lbacl *SCachedLoadbalancerAcl) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbacl *SCachedLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, true)
	if err != nil {
		return nil, err
	}
	return lbacl.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbacl *SCachedLoadbalancerAcl) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
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

func (lbacl *SCachedLoadbalancerAcl) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)

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

func (lbacl *SCachedLoadbalancerAcl) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := lbacl.GetDriver()
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

func (lbacl *SCachedLoadbalancerAcl) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbacl.SSharableVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	providerInfo := lbacl.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if providerInfo != nil {
		extra.Update(providerInfo)
	}
	regionInfo := lbacl.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	return extra
}

func (lbacl *SCachedLoadbalancerAcl) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbacl.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbacl *SCachedLoadbalancerAcl) AllowPerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return lbacl.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lbacl, "patch")
}

// PerformPatch patches acl entries by adding then deleting the specified acls.
// This is intended mainly for command line operations.
func (lbacl *SCachedLoadbalancerAcl) PerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	aclEntries := gotypes.DeepCopy(*lbacl.AclEntries).(SLoadbalancerAclEntries)
	{
		adds := SLoadbalancerAclEntries{}
		addsV := validators.NewStructValidator("adds", &adds)
		addsV.Optional(true)
		err := addsV.Validate(data)
		if err != nil {
			return nil, err
		}
		for _, add := range adds {
			found := false
			for _, aclEntry := range aclEntries {
				if aclEntry.Cidr == add.Cidr {
					found = true
					aclEntry.Comment = add.Comment
					break
				}
			}
			if !found {
				aclEntries = append(aclEntries, add)
			}
		}
	}
	{
		dels := SLoadbalancerAclEntries{}
		delsV := validators.NewStructValidator("dels", &dels)
		delsV.Optional(true)
		err := delsV.Validate(data)
		if err != nil {
			return nil, err
		}
		for _, del := range dels {
			for i := len(aclEntries) - 1; i >= 0; i-- {
				aclEntry := aclEntries[i]
				if aclEntry.Cidr == del.Cidr {
					aclEntries = append(aclEntries[:i], aclEntries[i+1:]...)
					break
				}
			}
		}
	}
	diff, err := db.Update(lbacl, func() error {
		lbacl.AclEntries = &aclEntries
		return nil
	})
	if err != nil {
		return nil, err
	}
	db.OpsLog.LogEvent(lbacl, db.ACT_UPDATE, diff, userCred)
	return nil, nil
}

func (lbacl *SCachedLoadbalancerAcl) ValidateDeleteCondition(ctx context.Context) error {
	man := LoadbalancerListenerManager
	t := man.TableSpec().Instance()
	pdF := t.Field("pending_deleted")
	lbaclId := lbacl.Id
	n, err := t.Query().
		Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
		Equals("acl_id", lbaclId).
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

func (lbacl *SCachedLoadbalancerAcl) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbacl, "purge")
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

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = self.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		self.DoPendingDelete(ctx, userCred)
	}
	return err
}

func (acl *SCachedLoadbalancerAcl) SyncWithCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, extAcl cloudprovider.ICloudLoadbalancerAcl, projectId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, acl, func() error {
		// todo: 华为云acl没有name字段应此不需要同步名称
		if api.CLOUD_PROVIDER_HUAWEI != acl.GetProviderName() {
			acl.Name = extAcl.GetName()
		} else {
			ext_listener_id := extAcl.GetAclListenerID()
			if len(ext_listener_id) > 0 {
				ilistener, err := db.FetchByExternalId(LoadbalancerListenerManager, ext_listener_id)
				if err != nil {
					return err
				}

				acl.ListenerId = ilistener.(*SLoadbalancerListener).GetId()
			}
		}

		acl.AclEntries = &SLoadbalancerAclEntries{}
		for _, entry := range extAcl.GetAclEntries() {
			*acl.AclEntries = append(*acl.AclEntries, &SLoadbalancerAclEntry{Cidr: entry.CIDR, Comment: entry.Comment})
		}
		return nil
	})
	if err != nil {
		return err
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
	if lblis.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI {
		listenerId = lblis.Id
	}

	lbacl, err := man.getLoadbalancerAclByRegion(provider, lblis.CloudregionId, acl.Id, listenerId)
	if err == nil {
		if lbacl.Id != acl.Id {
			_, err := man.TableSpec().Update(&lbacl, func() error {
				lbacl.Name = acl.Name
				lbacl.AclId = acl.Id
				lbacl.AclEntries = acl.AclEntries
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
	lbacl.ManagerId = lblis.ManagerId
	lbacl.CloudregionId = lblis.CloudregionId
	lbacl.ProjectId = lblis.ProjectId
	lbacl.ProjectSrc = lblis.ProjectSrc
	lbacl.Name = acl.Name
	lbacl.AclId = acl.Id
	lbacl.AclEntries = acl.AclEntries
	lbacl.ListenerId = listenerId

	err = man.TableSpec().Insert(&lbacl)
	if err != nil {
		return nil, err
	}

	lbacl, err = man.getLoadbalancerAclByRegion(provider, lblis.CloudregionId, acl.Id, listenerId)
	if err == nil {
		return &lbacl, nil
	}

	return nil, err
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
	ownerProjId := provider.ProjectId

	lockman.LockClass(ctx, man, ownerProjId)
	defer lockman.ReleaseClass(ctx, man, ownerProjId)

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

	newName, err := db.GenerateName(man, projectId, extAcl.GetName())
	if err != nil {
		return nil, err
	}
	acl.ExternalId = extAcl.GetGlobalId()
	acl.Name = newName
	acl.ManagerId = provider.Id
	acl.CloudregionId = region.Id

	acl.AclEntries = &SLoadbalancerAclEntries{}
	for _, entry := range extAcl.GetAclEntries() {
		*acl.AclEntries = append(*acl.AclEntries, &SLoadbalancerAclEntry{Cidr: entry.CIDR, Comment: entry.Comment})
	}

	f := acl.AclEntries.Fingerprint()
	if LoadbalancerAclManager.CountByFingerPrint(f) == 0 {
		localAcl := SLoadbalancerAcl{}
		localAcl.Name = acl.Name
		localAcl.Description = acl.Description
		localAcl.AclEntries = acl.AclEntries
		localAcl.Fingerprint = f
		// usercread
		localAcl.DomainId = userCred.GetProjectDomainId()
		localAcl.ProjectId = userCred.GetProjectId()
		localAcl.ProjectSrc = string(db.PROJECT_SOURCE_CLOUD)
		err := LoadbalancerAclManager.TableSpec().Insert(&localAcl)
		if err != nil {
			return nil, err
		}
	}

	{
		localAcl, err := LoadbalancerAclManager.FetchByFingerPrint(f)
		if err != nil {
			return nil, err
		}

		acl.AclId = localAcl.GetId()
	}

	err = man.TableSpec().Insert(&acl)
	if err != nil {
		log.Errorf("newFromCloudLoadbalancerAcl fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &acl, projectId, extAcl, acl.ManagerId)

	db.OpsLog.LogEvent(&acl, db.ACT_CREATE, acl.GetShortDesc(ctx), userCred)

	return &acl, nil
}

func (manager *SCachedLoadbalancerAclManager) InitializeData() error {
	// todo: sync old data from acls
	return nil
}
