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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerAclManager struct {
	SLoadbalancerLogSkipper

	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager

	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var LoadbalancerAclManager *SLoadbalancerAclManager

func init() {
	LoadbalancerAclManager = &SLoadbalancerAclManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLoadbalancerAcl{},
			"loadbalanceracls_tbl",
			"loadbalanceracl",
			"loadbalanceracls",
		),
	}
	LoadbalancerAclManager.SetVirtualObject(LoadbalancerAclManager)
}

type SLoadbalancerAcl struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	AclEntries  *api.SAclEntries `list:"user" update:"user" create:"required"`
	Fingerprint string           `name:"fingerprint" width:"64" charset:"ascii" nullable:"false" index:"true" list:"user" update:"user" create:"required"`
	// 是否变化
	IsDirty bool `nullable:"false" default:"false"`
}

func (man *SLoadbalancerAclManager) FetchByFingerPrint(projectId string, fingerprint string) (*SLoadbalancerAcl, error) {
	ret := &SLoadbalancerAcl{}
	q := man.Query().Equals("tenant_id", projectId).Equals("fingerprint", fingerprint).Asc("created_at").Limit(1)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (man *SLoadbalancerAclManager) CountByFingerPrint(projectId string, fingerprint string) int {
	q := man.Query()
	return q.Equals("tenant_id", projectId).Equals("fingerprint", fingerprint).Asc("created_at").Count()
}

func (man *SLoadbalancerAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LoadbalancerAclCreateInput) (*api.LoadbalancerAclCreateInput, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	input.Status = api.LB_STATUS_ENABLED
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (lbacl *SLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerAclUpdateInput) (*api.LoadbalancerAclUpdateInput, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}

	input.SharableVirtualResourceBaseUpdateInput, err = lbacl.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}

	if len(input.Fingerprint) > 0 && input.Fingerprint != lbacl.Fingerprint {
		db.Update(lbacl, func() error {
			lbacl.IsDirty = true
			return nil
		})
	}

	return input, nil
}

func (lbacl *SLoadbalancerAcl) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	if lbacl.IsDirty {
		acls, err := lbacl.GetCachedAcls()
		if err != nil {
			log.Errorf("SLoadbalancerAcl PostUpdate %s", err)
		}

		for i := range acls {
			acl := acls[i]
			acl.SetModelManager(CachedLoadbalancerAclManager, &acl)
			err = acl.StartLoadBalancerAclSyncTask(ctx, userCred, "")
			if err != nil {
				log.Errorf("SLoadbalancerAcl PostUpdate %s", err)
			}
		}
		db.Update(lbacl, func() error {
			lbacl.IsDirty = false
			return nil
		})
	}
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclSyncTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "LoadbalancerAclSyncTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SLoadbalancerAclManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerAclDetails {
	rows := make([]api.LoadbalancerAclDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerAclDetails{
			SharableVirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:            managerRows[i],
			CloudregionResourceInfo:        regionRows[i],
		}
	}

	for i := range objs {
		q := LoadbalancerListenerManager.Query().Equals("acl_id", objs[i].(*SLoadbalancerAcl).GetId())
		ownerId, queryScope, err, _ := db.FetchCheckQueryOwnerScope(ctx, userCred, query, LoadbalancerListenerManager, policy.PolicyActionList, true)
		if err != nil {
			log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
			return rows
		}

		q = LoadbalancerListenerManager.FilterByOwner(q, LoadbalancerListenerManager, userCred, ownerId, queryScope)
		rows[i].LbListenerCount, _ = q.CountWithError()
	}

	return rows
}

// PerformPatch patches acl entries by adding then deleting the specified acls.
// This is intended mainly for command line operations.
func (lbacl *SLoadbalancerAcl) PerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerAclPatchInput) (*jsonutils.JSONDict, error) {
	entries := *lbacl.AclEntries
	{
		for i := range input.Adds {
			add := input.Adds[i]
			err := add.Validate()
			if err != nil {
				return nil, err
			}
			found := false
			for _, aclEntry := range entries {
				if aclEntry.Cidr == add.Cidr {
					found = true
					aclEntry.Comment = add.Comment
					break
				}
			}
			if !found {
				entries = append(entries, add)
			}
		}
	}
	{
		for i := range input.Dels {
			del := input.Dels[i]
			err := del.Validate()
			if err != nil {
				return nil, err
			}
			for j := len(entries) - 1; j >= 0; j-- {
				aclEntry := entries[j]
				if aclEntry.Cidr == del.Cidr {
					entries = append(entries[:j], entries[j+1:]...)
					break
				}
			}
		}
	}
	diff, err := db.Update(lbacl, func() error {
		// todo: sync diff to clouds
		lbacl.AclEntries = &entries
		lbacl.Fingerprint = lbacl.AclEntries.GetFingerprint()
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(diff) > 0 {
		db.OpsLog.LogEvent(lbacl, db.ACT_UPDATE, diff, userCred)
	}
	return nil, nil
}

func (lbacl *SLoadbalancerAcl) ValidateDeleteCondition(ctx context.Context, info *api.LoadbalancerAclDetails) error {
	if info != nil && info.LbListenerCount > 0 {
		return httperrors.NewResourceBusyError("acl %s is still referred to by %d listener", lbacl.Name, info.LbListenerCount)
	}
	return lbacl.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, jsonutils.Marshal(info))
}

func (lbacl *SLoadbalancerAcl) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lbacl.Delete(ctx, userCred)
}

func (lbacl *SLoadbalancerAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	caches, err := lbacl.GetCachedAcls()
	if err != nil {
		return errors.Wrap(err, "GetCachedAcls")
	}

	for i := range caches {
		err = caches[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "RealDelete")
		}
	}

	return lbacl.DoPendingDelete(ctx, userCred)
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclDeleteTask", lbacl, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lbacl *SLoadbalancerAcl) GetCachedAcls() ([]SCachedLoadbalancerAcl, error) {
	ret := []SCachedLoadbalancerAcl{}
	q := CachedLoadbalancerAclManager.Query().Equals("acl_id", lbacl.Id)
	err := db.FetchModelObjects(CachedLoadbalancerAclManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// 负载均衡ACL规则列表
func (manager *SLoadbalancerAclManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	if len(input.Fingerprint) > 0 {
		q = q.In("fingerprint", input.Fingerprint)
	}

	return q, nil
}

func (manager *SLoadbalancerAclManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SLoadbalancerAclManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerAclManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
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

	return q, nil
}

func (manager *SLoadbalancerAclManager) InitializeData() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set deleted = true where pending_deleted = true",
			manager.TableSpec().Name(),
		),
	)
	return err
}
