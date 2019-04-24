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
	"net"
	"reflect"
	"strings"
	"unicode"

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

type SLoadbalancerAclEntry struct {
	Cidr    string
	Comment string
}

func (aclEntry *SLoadbalancerAclEntry) Validate(data *jsonutils.JSONDict) error {
	if strings.Index(aclEntry.Cidr, "/") > 0 {
		_, ipNet, err := net.ParseCIDR(aclEntry.Cidr)
		if err != nil {
			return err
		}
		// normalize from 192.168.1.3/24 to 192.168.1.0/24
		aclEntry.Cidr = ipNet.String()
	} else {
		ip := net.ParseIP(aclEntry.Cidr).To4()
		if ip == nil {
			return httperrors.NewInputParameterError("invalid addr %s", aclEntry.Cidr)
		}
	}
	if commentLimit := 128; len(aclEntry.Comment) > commentLimit {
		return httperrors.NewInputParameterError("comment too long (%d>=%d)",
			len(aclEntry.Comment), commentLimit)
	}
	for _, r := range aclEntry.Comment {
		if !unicode.IsPrint(r) {
			return httperrors.NewInputParameterError("comment contains non-printable char: %v", r)
		}
	}
	return nil
}

type SLoadbalancerAclEntries []*SLoadbalancerAclEntry

func (aclEntries *SLoadbalancerAclEntries) String() string {
	return jsonutils.Marshal(aclEntries).String()
}
func (aclEntries *SLoadbalancerAclEntries) IsZero() bool {
	if len([]*SLoadbalancerAclEntry(*aclEntries)) == 0 {
		return true
	}
	return false
}

func (aclEntries *SLoadbalancerAclEntries) Validate(data *jsonutils.JSONDict) error {
	found := map[string]bool{}
	for _, aclEntry := range *aclEntries {
		if err := aclEntry.Validate(data); err != nil {
			return err
		}
		if _, ok := found[aclEntry.Cidr]; ok {
			// error so that the user has a chance to deal with comments
			return httperrors.NewInputParameterError("acl cidr duplicate %s", aclEntry.Cidr)
		}
		found[aclEntry.Cidr] = true
	}
	return nil
}

type SLoadbalancerAclManager struct {
	SLoadbalancerLogSkipper
	db.SSharableVirtualResourceBaseManager
}

var LoadbalancerAclManager *SLoadbalancerAclManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SLoadbalancerAclEntries{}), func() gotypes.ISerializable {
		return &SLoadbalancerAclEntries{}
	})
	LoadbalancerAclManager = &SLoadbalancerAclManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLoadbalancerAcl{},
			"loadbalanceracls_tbl",
			"loadbalanceracl",
			"loadbalanceracls",
		),
	}
}

type SLoadbalancerAcl struct {
	db.SSharableVirtualResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	AclEntries *SLoadbalancerAclEntries `list:"user" update:"user" create:"required"`
}

func loadbalancerAclsValidateAclEntries(data *jsonutils.JSONDict, update bool) (*jsonutils.JSONDict, error) {
	aclEntries := SLoadbalancerAclEntries{}
	aclEntriesV := validators.NewStructValidator("acl_entries", &aclEntries)
	if update {
		aclEntriesV.Optional(true)
	}
	err := aclEntriesV.Validate(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (man *SLoadbalancerAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, false)
	if err != nil {
		return nil, err
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}

	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", "")
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

func (lbacl *SLoadbalancerAcl) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbacl *SLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, true)
	if err != nil {
		return nil, err
	}
	return lbacl.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbacl *SLoadbalancerAcl) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	lbacl.SetStatus(userCred, api.LB_SYNC_CONF, "")
	lbacl.StartLoadBalancerAclSyncTask(ctx, userCred, "")
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclSyncTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbacl *SLoadbalancerAcl) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)

	lbacl.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbacl.StartLoadBalancerAclCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalanceracl error: %v", err)
	}
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclCreateTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbacl *SLoadbalancerAcl) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(lbacl.CloudregionId)
	if err != nil {
		log.Errorf("failed to find region for loadbalancer acl %s", lbacl.Name)
		return nil
	}
	return region.(*SCloudregion)
}

func (lbacl *SLoadbalancerAcl) GetIRegion() (cloudprovider.ICloudRegion, error) {
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

func (lbacl *SLoadbalancerAcl) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
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

func (lbacl *SLoadbalancerAcl) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbacl.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbacl *SLoadbalancerAcl) AllowPerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return lbacl.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lbacl, "patch")
}

// PerformPatch patches acl entries by adding then deleting the specified acls.
// This is intended mainly for command line operations.
func (lbacl *SLoadbalancerAcl) PerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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

func (lbacl *SLoadbalancerAcl) ValidateDeleteCondition(ctx context.Context) error {
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

func (lbacl *SLoadbalancerAcl) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbacl, "purge")
}

func (lbacl *SLoadbalancerAcl) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbacl.StartLoadBalancerAclDeleteTask(ctx, userCred, parasm, "")
}

func (lbacl *SLoadbalancerAcl) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbacl.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lbacl.StartLoadBalancerAclDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclDeleteTask", lbacl, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbacl *SLoadbalancerAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerAclManager) getLoadbalancerAclsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SLoadbalancerAcl, error) {
	acls := []SLoadbalancerAcl{}
	q := man.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &acls); err != nil {
		log.Errorf("failed to get acls for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return acls, nil
}

func (man *SLoadbalancerAclManager) SyncLoadbalancerAcls(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, acls []cloudprovider.ICloudLoadbalancerAcl, syncRange *SSyncRange) compare.SyncResult {
	ownerProjId := provider.ProjectId

	lockman.LockClass(ctx, man, ownerProjId)
	defer lockman.ReleaseClass(ctx, man, ownerProjId)

	syncResult := compare.SyncResult{}

	dbAcls, err := man.getLoadbalancerAclsByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerAcl{}
	commondb := []SLoadbalancerAcl{}
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
		err = commondb[i].SyncWithCloudLoadbalancerAcl(ctx, userCred, commonext[i], provider.ProjectId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerAcl(ctx, userCred, provider, added[i], region, ownerProjId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SLoadbalancerAcl) syncRemoveCloudLoadbalanceAcl(ctx context.Context, userCred mcclient.TokenCredential) error {
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

func (man *SLoadbalancerAclManager) newFromCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extAcl cloudprovider.ICloudLoadbalancerAcl, region *SCloudregion, projectId string) (*SLoadbalancerAcl, error) {
	acl := SLoadbalancerAcl{}
	acl.SetModelManager(man)

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
	err = man.TableSpec().Insert(&acl)
	if err != nil {
		log.Errorf("newFromCloudLoadbalancerAcl fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &acl, projectId, extAcl, acl.ManagerId)

	db.OpsLog.LogEvent(&acl, db.ACT_CREATE, acl.GetShortDesc(ctx), userCred)

	return &acl, nil
}

func (acl *SLoadbalancerAcl) SyncWithCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, extAcl cloudprovider.ICloudLoadbalancerAcl, projectId string) error {
	diff, err := db.UpdateWithLock(ctx, acl, func() error {
		acl.Name = extAcl.GetName()
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
