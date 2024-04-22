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
type SCloudgroupPolicyManager struct {
	SCloudgroupJointsManager
	SCloudpolicyResourceBaseManager
}

var CloudgroupPolicyManager *SCloudgroupPolicyManager

func init() {
	db.InitManager(func() {
		CloudgroupPolicyManager = &SCloudgroupPolicyManager{
			SCloudgroupJointsManager: NewCloudgroupJointsManager(
				SCloudgroupPolicy{},
				"cloudgroup_policies_tbl",
				"cloudgroup_policy",
				"cloudgroup_policies",
				CloudpolicyManager,
			),
		}
		CloudgroupPolicyManager.SetVirtualObject(CloudgroupPolicyManager)
		CloudgroupPolicyManager.TableSpec().AddIndex(true, "cloudgroup_id", "cloudpolicy_id")
	})

}

// +onecloud:swagger-gen-ignore
type SCloudgroupPolicy struct {
	SCloudgroupJointsBase
	SCloudpolicyResourceBase
}

func (manager *SCloudgroupPolicyManager) GetSlaveFieldName() string {
	return "cloudpolicy_id"
}

// +onecloud:swagger-gen-ignore
func (manager *SCloudgroupPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotSupportedError("Not Supported")
}

// +onecloud:swagger-gen-ignore
func (self *SCloudgroupPolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SCloudgroupJointsBase.CustomizeDelete(ctx, userCred, query, data)
}

// +onecloud:swagger-gen-ignore
func (self *SCloudgroupPolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotSupportedError("Not Supported")
}

// 用户组中权限详情
func (manager *SCloudgroupPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudgroupPolicyDetails {
	rows := make([]api.CloudgroupPolicyDetails, len(objs))

	groupRows := manager.SCloudgroupJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	policyRows := manager.SCloudpolicyResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.CloudgroupPolicyDetails{
			CloudgroupJointResourceDetails: groupRows[i],
			CloudpolicyResourceDetails:     policyRows[i],
		}
	}

	return rows
}

func (self *SCloudgroupPolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SCloudgroupPolicy) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

// 用户组中权限列表
func (manager *SCloudgroupPolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudgroupPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SCloudgroupJointsManager.ListItemFilter(ctx, q, userCred, query.CloudgroupJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudgroupJointsManager.ListItemFilter")
	}

	q, err = manager.SCloudpolicyResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudpolicyResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SCloudgroupPolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudgroupPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SCloudgroupJointsManager.OrderByExtraFields(ctx, q, userCred, query.CloudgroupJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudgroupJointsManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SCloudgroupPolicyManager) ListItemExportKeys(ctx context.Context,
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
