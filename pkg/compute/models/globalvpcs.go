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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGlobalVpcManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
}

var GlobalVpcManager *SGlobalVpcManager

func init() {
	GlobalVpcManager = &SGlobalVpcManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SGlobalVpc{},
			"globalvpcs_tbl",
			"globalvpc",
			"globalvpcs",
		),
	}
	GlobalVpcManager.SetVirtualObject(GlobalVpcManager)
}

type SGlobalVpc struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
}

func (self *SGlobalVpc) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return httperrors.NewInternalServerError("GetVpcs fail %s", err)
	}
	if len(vpcs) > 0 {
		return httperrors.NewNotEmptyError("global vpc has associate %d vpcs", len(vpcs))
	}
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SGlobalVpc) GetVpcQuery() *sqlchemy.SQuery {
	return VpcManager.Query().Equals("globalvpc_id", self.Id)
}

func (self *SGlobalVpc) GetVpcs() ([]SVpc, error) {
	vpcs := []SVpc{}
	q := self.GetVpcQuery()
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, err
	}
	return vpcs, nil
}

func (self *SGlobalVpc) GetVpcCount() (int, error) {
	return self.GetVpcQuery().CountWithError()
}

func (manager *SGlobalVpcManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GlobalVpcDetails {
	rows := make([]api.GlobalVpcDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.GlobalVpcDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    managerRows[i],
		}
		gv := objs[i].(*SGlobalVpc)
		rows[i].VpcCount, _ = gv.GetVpcCount()
	}
	return rows
}

func (manager *SGlobalVpcManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.GlobalVpcCreateInput,
) (api.GlobalVpcCreateInput, error) {
	input.Status = apis.STATUS_CREATING
	var err error
	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData")
	}
	if len(input.CloudproviderId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudprovider_id")
	}
	_, err = validators.ValidateModel(userCred, CloudproviderManager, &input.CloudproviderId)
	if err != nil {
		return input, err
	}
	input.ManagerId = input.CloudproviderId
	quota := &SDomainQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{
			DomainId: ownerId.GetProjectDomainId(),
		},
		Globalvpc: 1,
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, errors.Wrap(err, "CheckSetPendingQuota")
	}
	return input, nil
}

func (self *SGlobalVpc) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	quota := &SDomainQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{
			DomainId: ownerId.GetProjectDomainId(),
		},
		Globalvpc: 1,
	}
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage %s", err)
	}
	self.StartCreateTask(ctx, userCred, "")
}

func (self *SGlobalVpc) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GlobalVpcCreateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SGlobalVpc) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GlobalvpcUpdateInput,
) (api.GlobalvpcUpdateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.ValidateUpdateData")
	}
	return input, nil
}

// 全局VPC列表
func (manager *SGlobalVpcManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GlobalVpcListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SGlobalVpcManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GlobalVpcListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	if db.NeedOrderQuery([]string{query.OrderByVpcCount}) {
		vpcQ := VpcManager.Query()
		vpcQ = vpcQ.AppendField(vpcQ.Field("globalvpc_id"), sqlchemy.COUNT("vpc_count"))
		vpcQ = vpcQ.GroupBy(vpcQ.Field("globalvpc_id"))
		vpcSQ := vpcQ.SubQuery()
		q = q.LeftJoin(vpcSQ, sqlchemy.Equals(vpcSQ.Field("globalvpc_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(vpcSQ.Field("vpc_count"))
		q = db.OrderByFields(q, []string{query.OrderByVpcCount}, []sqlchemy.IQueryField{q.Field("vpc_count")})
	}
	return q, nil
}

func (manager *SGlobalVpcManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SGlobalVpc) ValidateUpdateCondition(ctx context.Context) error {
	return self.SEnabledStatusInfrasResourceBase.ValidateUpdateCondition(ctx)
}

func (manager *SGlobalVpcManager) totalCount(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) int {
	q := manager.Query()
	switch scope {
	case rbacscope.ScopeProject, rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}
	cnt, _ := q.CountWithError()
	return cnt
}

func (globalVpc *SGlobalVpc) GetUsages() []db.IUsage {
	if globalVpc.Deleted {
		return nil
	}
	usage := SDomainQuota{Globalvpc: 1}
	usage.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: globalVpc.DomainId})
	return []db.IUsage{
		&usage,
	}
}

func (globalVpc *SGlobalVpc) GetRequiredSharedDomainIds() []string {
	vpcs, _ := globalVpc.GetVpcs()
	if len(vpcs) == 0 {
		return globalVpc.SEnabledStatusInfrasResourceBase.GetRequiredSharedDomainIds()
	}
	requires := make([][]string, len(vpcs))
	for i := range vpcs {
		requires[i] = db.ISharableChangeOwnerCandidateDomainIds(&vpcs[i])
	}
	return db.ISharableMergeShareRequireDomainIds(requires...)
}

func (globalVpc *SGlobalVpc) GetChangeOwnerRequiredDomainIds() []string {
	requires := stringutils2.SSortedStrings{}
	vpcs, _ := globalVpc.GetVpcs()
	for i := range vpcs {
		requires = stringutils2.Append(requires, vpcs[i].DomainId)
	}
	return requires
}

func (self *SCloudprovider) GetGlobalVpcs() ([]SGlobalVpc, error) {
	q := GlobalVpcManager.Query().Equals("manager_id", self.Id)
	vpcs := []SGlobalVpc{}
	err := db.FetchModelObjects(GlobalVpcManager, q, &vpcs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return vpcs, nil
}

func (self *SCloudprovider) SyncGlobalVpcs(ctx context.Context, userCred mcclient.TokenCredential, exts []cloudprovider.ICloudGlobalVpc, xor bool) ([]SGlobalVpc, []cloudprovider.ICloudGlobalVpc, compare.SyncResult) {
	lockman.LockRawObject(ctx, GlobalVpcManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, GlobalVpcManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	localVpcs := make([]SGlobalVpc, 0)
	remoteVpcs := make([]cloudprovider.ICloudGlobalVpc, 0)

	dbVpcs, err := self.GetGlobalVpcs()
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	removed := make([]SGlobalVpc, 0)
	commondb := make([]SGlobalVpc, 0)
	commonext := make([]cloudprovider.ICloudGlobalVpc, 0)
	added := make([]cloudprovider.ICloudGlobalVpc, 0)

	err = compare.CompareSets(dbVpcs, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveGlobalVpc(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudGlobalVpc(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localVpcs = append(localVpcs, commondb[i])
		remoteVpcs = append(remoteVpcs, commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		vpc, err := self.newFromCloudGlobalVpc(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		localVpcs = append(localVpcs, *vpc)
		remoteVpcs = append(remoteVpcs, added[i])
		result.Add()
	}
	return localVpcs, remoteVpcs, result
}

func (self *SGlobalVpc) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SGlobalVpc) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SGlobalVpc) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SGlobalVpc) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GlobalVpcDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SGlobalVpc) GetICloudGlobalVpc(ctx context.Context) (cloudprovider.ICloudGlobalVpc, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	return provider.GetICloudGlobalVpcById(self.ExternalId)
}

func (self *SGlobalVpc) syncRemoveGlobalVpc(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		self.SetStatus(userCred, apis.STATUS_UNKNOWN, "sync remove")
		return err
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SGlobalVpc) SyncWithCloudGlobalVpc(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudGlobalVpc) error {
	_, err := db.Update(self, func() error {
		self.Status = ext.GetStatus()
		return nil
	})
	return err
}

func (self *SCloudprovider) newFromCloudGlobalVpc(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudGlobalVpc) (*SGlobalVpc, error) {
	gvpc := &SGlobalVpc{}
	gvpc.SetModelManager(GlobalVpcManager, gvpc)
	gvpc.Name = ext.GetName()
	gvpc.Status = ext.GetStatus()
	gvpc.ExternalId = ext.GetGlobalId()
	gvpc.ManagerId = self.Id
	gvpc.DomainId = self.DomainId
	gvpc.Enabled = tristate.True

	return gvpc, GlobalVpcManager.TableSpec().Insert(ctx, gvpc)
}

// 同步全局VPC状态
func (self *SGlobalVpc) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.SyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Globalvpc has %d task active, can't sync status", count)
	}

	return nil, self.StartSyncstatusTask(ctx, userCred, "")
}

func (self *SGlobalVpc) StartSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "GlobalVpcSyncstatusTask", parentTaskId)
}

func (self *SGlobalVpc) GetSecgroups() ([]SSecurityGroup, error) {
	q := SecurityGroupManager.Query().Equals("globalvpc_id", self.Id)
	ret := []SSecurityGroup{}
	return ret, db.FetchModelObjects(SecurityGroupManager, q, &ret)
}

func (self *SGlobalVpc) SyncSecgroups(ctx context.Context, userCred mcclient.TokenCredential, exts []cloudprovider.ICloudSecurityGroup, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, SecurityGroupManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, SecurityGroupManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	dbSecs, err := self.GetSecgroups()
	if err != nil {
		result.Error(err)
		return result
	}

	provider := self.GetCloudprovider()

	syncOwnerId := provider.GetOwnerId()

	removed := make([]SSecurityGroup, 0)
	commondb := make([]SSecurityGroup, 0)
	commonext := make([]cloudprovider.ICloudSecurityGroup, 0)
	added := make([]cloudprovider.ICloudSecurityGroup, 0)

	err = compare.CompareSets(dbSecs, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].SyncWithCloudSecurityGroup(ctx, userCred, commonext[i], syncOwnerId, true)
			if err != nil {
				result.UpdateError(err)
				continue
			}
		}
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		err := self.newFromCloudSecurityGroup(ctx, userCred, added[i], syncOwnerId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SGlobalVpc) newFromCloudSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ext cloudprovider.ICloudSecurityGroup,
	syncOwnerId mcclient.IIdentityProvider,
) error {
	ret := &SSecurityGroup{}
	ret.SetModelManager(SecurityGroupManager, ret)
	ret.Name = ext.GetName()
	ret.Description = ext.GetDescription()
	ret.ExternalId = ext.GetGlobalId()
	ret.ManagerId = self.ManagerId
	ret.GlobalvpcId = self.Id
	ret.Status = api.SECGROUP_STATUS_READY
	err := SecurityGroupManager.TableSpec().Insert(ctx, ret)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}
	db.Update(ret, func() error {
		ret.CloudregionId = "-"
		return nil
	})

	syncVirtualResourceMetadata(ctx, userCred, ret, ext)
	SyncCloudProject(ctx, userCred, ret, syncOwnerId, ext, ret.ManagerId)

	rules, err := ext.GetRules()
	if err != nil {
		return errors.Wrapf(err, "GetRules")
	}
	result := ret.SyncRules(ctx, userCred, rules)
	if result.IsError() {
		logclient.AddSimpleActionLog(ret, logclient.ACT_CLOUD_SYNC, result, userCred, false)
	}
	return nil
}
