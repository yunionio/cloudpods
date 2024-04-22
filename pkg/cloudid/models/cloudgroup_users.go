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
type SCloudgroupUserManager struct {
	SCloudgroupJointsManager
	SClouduserResourceBaseManager
}

var CloudgroupUserManager *SCloudgroupUserManager

func init() {
	db.InitManager(func() {
		CloudgroupUserManager = &SCloudgroupUserManager{
			SCloudgroupJointsManager: NewCloudgroupJointsManager(
				SCloudgroupUser{},
				"cloudgroup_users_tbl",
				"cloudgroup_user",
				"cloudgroup_users",
				ClouduserManager,
			),
		}
		CloudgroupUserManager.SetVirtualObject(CloudgroupUserManager)
		CloudgroupUserManager.TableSpec().AddIndex(true, "cloudgroup_id", "clouduser_id")
	})

}

// +onecloud:swagger-gen-ignore
type SCloudgroupUser struct {
	SCloudgroupJointsBase

	SClouduserResourceBase
}

func (manager *SCloudgroupUserManager) GetSlaveFieldName() string {
	return "clouduser_id"
}

// +onecloud:swagger-gen-ignore
func (manager *SCloudgroupUserManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotSupportedError("Not Supported")
}

// +onecloud:swagger-gen-ignore
func (self *SCloudgroupUser) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SCloudgroupJointsBase.CustomizeDelete(ctx, userCred, query, data)
}

// +onecloud:swagger-gen-ignore
func (self *SCloudgroupUser) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotSupportedError("Not Supported")
}

// 获取用户组中用户详情
func (manager *SCloudgroupUserManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudgroupUserDetails {
	rows := make([]api.CloudgroupUserDetails, len(objs))

	groupRows := manager.SCloudgroupJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userRows := manager.SClouduserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.CloudgroupUserDetails{
			CloudgroupJointResourceDetails: groupRows[i],
			ClouduserResourceDetails:       userRows[i],
		}
	}

	return rows
}

func (self *SCloudgroupUser) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SCloudgroupUser) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

// 用户组中用户列表
func (manager *SCloudgroupUserManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudgroupUserListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SCloudgroupJointsManager.ListItemFilter(ctx, q, userCred, query.CloudgroupJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudgroupJointsManager.ListItemFilter")
	}

	q, err = manager.SClouduserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ClouduserResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SCloudgroupUserManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudgroupUserListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SCloudgroupJointsManager.OrderByExtraFields(ctx, q, userCred, query.CloudgroupJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudgroupJointsManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SCloudgroupUserManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SCloudgroupJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudgroupJointsManager.ListItemExportKeys")
	}

	return q, nil
}

/*
func (manager *SCloudgroupUserManager) newFromCloudgroupUser(ctx context.Context, userCred mcclient.TokenCredential, iUser cloudprovider.IClouduser, group *SCloudgroup) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	user := SCloudgroupUser{}
	user.SetModelManager(manager, &user)
	user.CloudgroupId = group.Id
	u, err := group.GetClouduserByExternalId(iUser.GetGlobalId())
	if err != nil {
		return errors.Wrapf(err, "GetClouduserByExternalId(%s)", iUser.GetGlobalId())
	}
	user.ClouduserId = u.GetId()
	return manager.TableSpec().Insert(&user)
}*/
