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
	OwnerId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
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
	return q, nil
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
	cnt, err := manager.Query().Equals("owner_id", input.OwnerId).Equals("cloudgroup_id", input.CloudgroupId).CountWithError()
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return input, httperrors.NewConflictError("user %s has already in group %s", input.Name, group.Name)
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
	for i := range rows {
		rows[i] = api.SamluserDetails{
			StatusDomainLevelUserResourceDetails: userRows[i],
			CloudgroupResourceDetails:            groupRows[i],
		}
	}
	return rows
}
