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

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudroleManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
	SCloudproviderResourceBaseManager
	SAMLProviderResourceBaseManager
	SCloudgroupResourceBaseManager
}

var CloudroleManager *SCloudroleManager

func init() {
	CloudroleManager = &SCloudroleManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SCloudrole{},
			"cloudroles_tbl",
			"cloudrole",
			"cloudroles",
		),
	}
	CloudroleManager.SetVirtualObject(CloudroleManager)
}

type SCloudrole struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SCloudaccountResourceBase
	SCloudproviderResourceBase
	SAMLProviderResourceBase
	SCloudgroupResourceBase

	Document *jsonutils.JSONDict `length:"long" charset:"ascii" list:"domain" update:"domain" create:"domain_required"`
}

// 公有云角色列表
func (manager *SCloudroleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudroleListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SCloudaccountResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudaccountResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SCloudproviderResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudproviderResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SCloudgroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudgroupResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

// 获取公有云角色详情
func (manager *SCloudroleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudroleDetails {
	rows := make([]api.CloudroleDetails, len(objs))
	infRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	mRows := manager.SCloudproviderResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.CloudroleDetails{
			StatusInfrasResourceBaseDetails: infRows[i],
			CloudaccountResourceDetails:     acRows[i],
			CloudproviderResourceDetails:    mRows[i],
		}
	}
	return rows
}

// 删除公有云角色
func (self *SCloudrole) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartCloudroleDeleteTask(ctx, userCred, "")
}

func (self *SCloudrole) StartCloudroleDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudroleDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETE_FAILED, "")
	return task.ScheduleRun(nil)
}

func (self *SCloudrole) GetCloudprovider() (*SCloudprovider, error) {
	provider, err := CloudproviderManager.FetchById(self.ManagerId)
	if err != nil {
		return nil, err
	}
	return provider.(*SCloudprovider), nil
}

func (self *SCloudrole) GetProvider() (cloudprovider.ICloudProvider, error) {
	if len(self.ManagerId) > 0 {
		provider, err := self.GetCloudprovider()
		if err != nil {
			return nil, err
		}
		return provider.GetProvider()
	}
	if len(self.CloudaccountId) > 0 {
		account, err := self.GetCloudaccount()
		if err != nil {
			if err != nil {
				return nil, err
			}
		}
		return account.GetProvider()
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty account info")
}

func (self *SCloudrole) GetICloudrole() (cloudprovider.ICloudrole, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	provider, err := self.GetProvider()
	if err != nil {
		return nil, err
	}
	return provider.GetICloudroleById(self.ExternalId)
}

func (self *SCloudrole) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SCloudrole) syncWithCloudrole(ctx context.Context, userCred mcclient.TokenCredential, iRole cloudprovider.ICloudrole) error {
	_, err := db.Update(self, func() error {
		self.Name = self.GetName()
		self.Document = iRole.GetDocument()
		self.Status = apis.STATUS_AVAILABLE
		return nil
	})
	return err
}

func (self *SCloudaccount) SyncCloudroles(ctx context.Context, userCred mcclient.TokenCredential, exts []cloudprovider.ICloudrole, managerId string) compare.SyncResult {
	result := compare.SyncResult{}

	roles, err := self.GetCloudroles(managerId)
	if err != nil {
		result.Error(errors.Wrapf(err, "GetCloudroles"))
		return result
	}

	removed := make([]SCloudrole, 0)
	commondb := make([]SCloudrole, 0)
	commonext := make([]cloudprovider.ICloudrole, 0)
	added := make([]cloudprovider.ICloudrole, 0)

	err = compare.CompareSets(roles, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].syncWithCloudrole(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err := self.newCloudrole(ctx, userCred, added[i], managerId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SCloudaccount) newCloudrole(ctx context.Context, userCred mcclient.TokenCredential, iRole cloudprovider.ICloudrole, managerId string) error {
	role := &SCloudrole{}
	role.SetModelManager(CloudroleManager, role)
	role.Name = iRole.GetName()
	role.ExternalId = iRole.GetGlobalId()
	role.Document = iRole.GetDocument()
	if spId := iRole.GetSAMLProvider(); len(spId) > 0 {
		sp, _ := db.FetchByExternalIdAndManagerId(SAMLProviderManager, spId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("cloudaccount_id", self.Id)
		})
		if sp != nil {
			role.SAMLProviderId = sp.GetId()
		}
	}
	role.CloudaccountId = self.Id
	role.ManagerId = managerId
	role.Status = apis.STATUS_AVAILABLE
	return CloudroleManager.TableSpec().Insert(ctx, role)
}
