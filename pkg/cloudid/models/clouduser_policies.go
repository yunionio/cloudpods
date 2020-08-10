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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SClouduserPolicyManager struct {
	SClouduserJointsManager

	SCloudpolicyResourceBaseManager
}

var ClouduserPolicyManager *SClouduserPolicyManager

func init() {
	db.InitManager(func() {
		ClouduserPolicyManager = &SClouduserPolicyManager{
			SClouduserJointsManager: NewClouduserJointsManager(
				SClouduserPolicy{},
				"clouduser_policies_tbl",
				"clouduser_policy",
				"clouduser_policies",
				CloudpolicyManager,
			),
		}
		ClouduserPolicyManager.SetVirtualObject(ClouduserPolicyManager)
		ClouduserPolicyManager.TableSpec().AddIndex(true, "clouduser_id", "cloudpolicy_id")
	})

}

type SClouduserPolicy struct {
	SClouduserJointsBase

	SCloudpolicyResourceBase
}

func (manager *SClouduserPolicyManager) GetSlaveFieldName() string {
	return "cloudpolicy_id"
}

func (manager *SClouduserPolicyManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SClouduserPolicy) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

// +onecloud:swagger-gen-ignore
func (manager *SClouduserPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotSupportedError("Not Supported")
}

// +onecloud:swagger-gen-ignore
func (self *SClouduserPolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SClouduserJointsBase.CustomizeDelete(ctx, userCred, query, data)
}

// +onecloud:swagger-gen-ignore
func (self *SClouduserPolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotSupportedError("Not Supported")
}

// 获取公有云用户权限详情
func (self *SClouduserPolicy) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ClouduserPolicyDetails, error) {
	return api.ClouduserPolicyDetails{}, nil
}

func (manager *SClouduserPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ClouduserPolicyDetails {
	rows := make([]api.ClouduserPolicyDetails, len(objs))

	userRows := manager.SClouduserJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	policyRows := manager.SCloudpolicyResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.ClouduserPolicyDetails{
			ClouduserJointResourceDetails: userRows[i],
			CloudpolicyResourceDetails:    policyRows[i],
		}
	}

	return rows
}

func (self *SClouduserPolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SClouduserPolicy) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

// 公有云用户权限列表
func (manager *SClouduserPolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ClouduserPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SClouduserJointsManager.ListItemFilter(ctx, q, userCred, query.ClouduserJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SClouduserJointsManager.ListItemFilter")
	}
	q, err = manager.SCloudpolicyResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudpolicyResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SClouduserPolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ClouduserPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SClouduserJointsManager.OrderByExtraFields(ctx, q, userCred, query.ClouduserJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SClouduserJointsManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SClouduserPolicyManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SClouduserJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SClouduserJointsManager.ListItemExportKeys")
	}

	return q, nil
}

func (manager *SClouduserPolicyManager) newFromClouduserPolicy(ctx context.Context, userCred mcclient.TokenCredential, iPolicy cloudprovider.ICloudpolicy, user *SClouduser) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	account, err := user.GetCloudaccount()
	if err != nil {
		return errors.Wrap(err, "user.GetCloudaccount")
	}

	up := &SClouduserPolicy{}
	up.SetModelManager(manager, up)
	up.ClouduserId = user.Id

	p, err := db.FetchByExternalIdAndManagerId(CloudpolicyManager, iPolicy.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("provider", account.Provider)
	})

	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrapf(err, "db.FetchByExternalId(%s)", iPolicy.GetGlobalId())
		}
		policy, err := CloudpolicyManager.newFromCloudpolicy(ctx, userCred, iPolicy, account.Provider)
		if err != nil {
			return errors.Wrap(err, "newFromCloudpolicy")
		}
		up.CloudpolicyId = policy.Id
	} else {
		up.CloudpolicyId = p.GetId()
	}

	return manager.TableSpec().Insert(ctx, up)
}
