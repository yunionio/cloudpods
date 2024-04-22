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

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SClouduserPolicyManager struct {
	SClouduserJointsManager

	SCloudproviderResourceBaseManager
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
		ClouduserPolicyManager.TableSpec().AddIndex(false, "clouduser_id", "cloudpolicy_id")
	})

}

// +onecloud:swagger-gen-ignore
type SClouduserPolicy struct {
	SClouduserJointsBase

	SCloudpolicyResourceBase
}

func (manager *SClouduserPolicyManager) GetSlaveFieldName() string {
	return "cloudpolicy_id"
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

func (manager *SClouduserPolicyManager) InitializeData() error {
	return nil
}
