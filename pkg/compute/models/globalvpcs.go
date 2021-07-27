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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGlobalVpcManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
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
}

func (manager *SGlobalVpcManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SGlobalVpc) ValidateDeleteCondition(ctx context.Context) error {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrap(err, "self.GetVpcs")
	}
	if len(vpcs) > 0 {
		return fmt.Errorf("not an empty globalvpc")
	}
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
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
	for i := range rows {
		rows[i] = api.GlobalVpcDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
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
	input.Status = api.GLOBAL_VPC_STATUS_AVAILABLE
	var err error
	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData")
	}
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

func (manager *SGlobalVpcManager) totalCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) int {
	q := manager.Query()
	switch scope {
	case rbacutils.ScopeProject, rbacutils.ScopeDomain:
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
