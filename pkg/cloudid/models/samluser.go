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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSamluserManager struct {
	db.SStatusDomainLevelUserResourceBaseManager
	db.SExternalizedResourceBaseManager

	SCloudgroupResourceBaseManager
	SCloudaccountResourceBaseManager
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
	db.SExternalizedResourceBase
	SCloudgroupResourceBase
	SCloudaccountResourceBase

	// 邮箱地址
	Email string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"domain_optional"`
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
	accountId, _ := data.GetString("cloudaccount_id")
	return jsonutils.Marshal(map[string]string{"cloudgroup_id": groupId, "cloudaccount_id": accountId})
}

func (manager *SSamluserManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	groupId, _ := values.GetString("cloudgroup_id")
	accountId, _ := values.GetString("cloudaccount_id")
	if len(groupId) > 0 {
		q = q.Equals("cloudgroup_id", groupId)
	}
	if len(accountId) > 0 {
		q = q.Equals("cloudaccount_id", accountId)
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
	q, err = manager.SCloudaccountResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudaccountResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// 创建SAML认证用户
func (manager *SSamluserManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SamluserCreateInput) (api.SamluserCreateInput, error) {
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
	_group, err := validators.ValidateModel(userCred, CloudgroupManager, &input.CloudgroupId)
	if err != nil {
		return input, err
	}
	group := _group.(*SCloudgroup)
	_account, err := validators.ValidateModel(userCred, CloudaccountManager, &input.CloudaccountId)
	if err != nil {
		return input, err
	}
	account := _account.(*SCloudaccount)
	if account.SAMLAuth.IsFalse() {
		return input, httperrors.NewNotSupportedError("cloudaccount %s not enable saml auth", account.Name)
	}
	if account.Provider != group.Provider {
		return input, httperrors.NewConflictError("account %s and group %s not with same provider", account.Name, group.Name)
	}
	if account.Provider == compute_api.CLOUD_PROVIDER_AZURE {
		if info := strings.Split(options.Options.ApiServer, ":"); len(info) > 1 {
			domain := strings.TrimPrefix(info[1], "//")
			input.Email = fmt.Sprintf("%s@%s", input.Name, domain)
		}
	}
	sq := CloudgroupManager.Query("id").Equals("provider", group.Provider).SubQuery()
	q := manager.Query().Equals("owner_id", input.OwnerId).Equals("cloudaccount_id", account.Id).In("cloudgroup_id", sq)
	groups := []SCloudgroup{}
	err = db.FetchModelObjects(CloudgroupManager, q, &groups)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "db.FetchModelObjects"))
	}
	if len(groups) > 0 {
		return input, httperrors.NewConflictError("user %s has already in other %s group", input.Name, group.Provider)
	}
	input.Status = api.SAML_USER_STATUS_AVAILABLE
	return input, nil
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
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.SamluserDetails{
			StatusDomainLevelUserResourceDetails: userRows[i],
			CloudgroupResourceDetails:            groupRows[i],
			CloudaccountResourceDetails:          acRows[i],
		}
	}
	return rows
}

func (self *SSamluser) SyncAzureGroup() error {
	group, err := self.GetCloudgroup()
	if err != nil {
		return errors.Wrapf(err, "GetCloudgroup")
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccount")
	}
	cache, err := CloudgroupcacheManager.Register(group, account)
	if err != nil {
		return errors.Wrapf(err, "group cache Register")
	}
	if len(cache.ExternalId) == 0 {
		s := auth.GetAdminSession(context.TODO(), options.Options.Region)
		_, err = cache.GetOrCreateICloudgroup(context.TODO(), s.GetToken())
		if err != nil {
			return errors.Wrapf(err, "GetOrCreateICloudgroup")
		}
		cache, err = CloudgroupcacheManager.Register(group, account)
		if err != nil {
			return errors.Wrapf(err, "group cache Register")
		}
	}
	iGroup, err := cache.GetICloudgroup()
	if err != nil {
		return errors.Wrapf(err, "GetICloudgroup")
	}
	err = iGroup.AddUser(self.ExternalId)
	if err != nil {
		return errors.Wrapf(err, "iGroup.AddUser")
	}
	return nil
}
