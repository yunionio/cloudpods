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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSamluserManager struct {
	db.SStatusDomainLevelUserResourceBaseManager

	SCloudgroupResourceBaseManager
}

var SamluserManager *SSamluserManager

func init() {
	SamluserManager = &SSamluserManager{
		SStatusDomainLevelUserResourceBaseManager: db.NewStatusDomainLevelUserResourceBaseManager(
			SSamluser{},
			"samlusers_tbl",
			"samluser",
			"samlusers",
		),
	}
	SamluserManager.SetVirtualObject(SamluserManager)
}

type SSamluser struct {
	db.SStatusDomainLevelUserResourceBase
	SCloudgroupResourceBase

	// 邮箱地址
	Email string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`

	CloudroleId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SSamluserManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	q := manager.Query()
	domainCnt, err := db.CalculateResourceCount(q, "domain_id")
	if err != nil {
		return nil, errors.Wrap(err, "CalculateResourceCount.domain_id")
	}
	q = manager.Query()
	userCnt, err := db.CalculateResourceCount(q, "owner_id")
	if err != nil {
		return nil, errors.Wrap(err, "CalculateResourceCount.owner_id")
	}
	return append(domainCnt, userCnt...), nil
}

func (manager *SSamluserManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

func (manager *SSamluserManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	groupId, _ := data.GetString("cloudgroup_id")
	return jsonutils.Marshal(map[string]string{"cloudgroup_id": groupId})
}

func (manager *SSamluserManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	groupId, _ := values.GetString("cloudgroup_id")
	if len(groupId) > 0 {
		q = q.Equals("cloudgroup_id", groupId)
	}
	return q
}

// SAML认证用户列表
func (manager *SSamluserManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.SamluserListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusDomainLevelUserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusDomainLevelUserResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SCloudgroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudgroupResourceListInput)
	if err != nil {
		return nil, err
	}

	if len(query.CloudaccountId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, CloudaccountManager, &query.CloudaccountId)
		if err != nil {
			return nil, err
		}
		sq := CloudgroupManager.Query("id").Equals("cloudaccount_id", query.CloudaccountId)
		q = q.In("cloudgroup_id", sq.SubQuery())
	}

	return q, nil
}

func (manager *SSamluserManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusDomainLevelUserResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	switch field {
	case "manager":
		managerQuery := CloudproviderManager.Query("name", "id").SubQuery()
		groupQuery := CloudgroupManager.Query("id").SubQuery()
		q.AppendField(managerQuery.Field("name", field)).Distinct()
		q = q.Join(groupQuery, sqlchemy.Equals(q.Field("cloudgroup_id"), groupQuery.Field("id")))
		q = q.Join(managerQuery, sqlchemy.Equals(groupQuery.Field("manager_id"), managerQuery.Field("id")))
		return q, nil
	case "account":
		accountQuery := CloudaccountManager.Query("name", "id").SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		groupQuery := CloudgroupManager.Query("id").SubQuery()
		q.AppendField(accountQuery.Field("name", field)).Distinct()
		q = q.Join(groupQuery, sqlchemy.Equals(q.Field("cloudgroup_id"), groupQuery.Field("id")))
		q = q.Join(providers, sqlchemy.Equals(groupQuery.Field("manager_id"), providers.Field("id")))
		q = q.Join(accountQuery, sqlchemy.Equals(providers.Field("cloudaccount_id"), accountQuery.Field("id")))
		return q, nil
	case "provider", "brand":
		accountQuery := CloudaccountManager.Query(field, "id").Distinct().SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		groupQuery := CloudgroupManager.Query("id").SubQuery()
		q.AppendField(accountQuery.Field(field)).Distinct()
		q = q.Join(groupQuery, sqlchemy.Equals(q.Field("cloudgroup_id"), groupQuery.Field("id")))
		q = q.Join(providers, sqlchemy.Equals(groupQuery.Field("manager_id"), providers.Field("id")))
		q = q.Join(accountQuery, sqlchemy.Equals(providers.Field("cloudaccount_id"), accountQuery.Field("id")))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

// 创建SAML认证用户
func (manager *SSamluserManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.SamluserCreateInput,
) (*api.SamluserCreateInput, error) {
	if len(input.OwnerId) > 0 {
		user, err := db.UserCacheManager.FetchUserById(ctx, input.OwnerId)
		if err != nil {
			return input, errors.Wrapf(err, "FetchUserById")
		}
		input.OwnerId = user.Id
		if len(input.Name) == 0 {
			input.Name = user.Name
		}
	} else {
		input.OwnerId = userCred.GetUserId()
		if len(input.Name) == 0 {
			input.Name = userCred.GetUserName()
		}
	}
	groupObj, err := validators.ValidateModel(ctx, userCred, CloudgroupManager, &input.CloudgroupId)
	if err != nil {
		return input, err
	}
	group := groupObj.(*SCloudgroup)
	sq := CloudgroupManager.Query("id").Equals("cloudaccount_id", group.CloudaccountId)
	if len(group.ManagerId) > 0 {
		sq = sq.Equals("manager_id", group.ManagerId)
	}
	cnt, err := manager.Query().Equals("owner_id", input.OwnerId).In("cloudgroup_id", sq.SubQuery()).CountWithError()
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return input, httperrors.NewConflictError("user %s has already in group %s account", input.Name, group.Name)
	}
	input.Status = apis.STATUS_CREATING
	return input, nil
}

func (self *SSamluser) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.StartCreateTask(ctx, userCred, "")
}

func (self *SSamluser) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "SamlUserCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SSamluserManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SamluserDetails {
	rows := make([]api.SamluserDetails, len(objs))
	userRows := manager.SStatusDomainLevelUserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	groupRows := manager.SCloudgroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	groupIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.SamluserDetails{
			StatusDomainLevelUserResourceDetails: userRows[i],
			CloudgroupResourceDetails:            groupRows[i],
		}
		user := objs[i].(*SSamluser)
		groupIds[i] = user.CloudgroupId
	}

	groups := make(map[string]SCloudgroup)
	err := db.FetchStandaloneObjectsByIds(CloudgroupManager, groupIds, &groups)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	managerIds := make([]string, len(objs))
	accountIds := make([]string, len(objs))
	for i := range rows {
		if group, ok := groups[groupIds[i]]; ok {
			rows[i].CloudaccountId = group.CloudaccountId
			rows[i].ManagerId = group.ManagerId
			managerIds[i] = group.ManagerId
			accountIds[i] = group.CloudaccountId
		}
	}

	accounts := make(map[string]SCloudaccount)
	err = db.FetchStandaloneObjectsByIds(CloudaccountManager, accountIds, &accounts)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds for accounts fail %s", err)
		return nil
	}

	managers := make(map[string]SCloudprovider)
	err = db.FetchStandaloneObjectsByIds(CloudproviderManager, managerIds, &managers)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds for accounts fail %s", err)
		return nil
	}

	for i := range rows {
		if account, ok := accounts[rows[i].CloudaccountId]; ok {
			rows[i].Cloudaccount = account.Name
			rows[i].Provider = account.Provider
			rows[i].Brand = account.Brand
		}
		if provider, ok := managers[rows[i].ManagerId]; ok {
			rows[i].Manager = provider.Name
		}
	}

	return rows
}
