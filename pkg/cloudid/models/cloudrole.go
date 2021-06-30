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
	"strconv"
	"strings"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudroleManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
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
	SAMLProviderResourceBase
	SCloudgroupResourceBase

	Document *jsonutils.JSONDict `length:"long" charset:"ascii" list:"domain" update:"domain" create:"domain_required"`
	OwnerId  string              `width:"128" charset:"ascii" index:"true" list:"user" nullable:"false" create:"optional"`
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
	for i := range rows {
		rows[i] = api.CloudroleDetails{
			StatusInfrasResourceBaseDetails: infRows[i],
			CloudaccountResourceDetails:     acRows[i],
		}
	}
	return rows
}

// 删除公有云角色
func (self *SCloudrole) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartCloudroleDeleteTask(ctx, userCred, false, "")
}

func (self *SCloudrole) StartCloudroleDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, purge bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(purge), "purge")
	task, err := taskman.TaskManager.NewTask(ctx, "CloudroleDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.CLOUD_ROLE_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudrole) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "purge")
}

// 清除角色(不删除云上资源)
func (self *SCloudrole) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudrolePurgeInput) (jsonutils.JSONObject, error) {
	return nil, self.StartCloudroleDeleteTask(ctx, userCred, true, "")
}

func (self *SCloudrole) GetICloudrole() (cloudprovider.ICloudrole, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
	}
	provider, err := account.GetProvider()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	if len(self.ExternalId) > 0 {
		iRole, err := provider.GetICloudroleById(self.ExternalId)
		if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, errors.Wrapf(err, "GetICloudroleById(%s)", self.ExternalId)
		}
		if err == nil {
			return iRole, nil
		}
	}
	sp, err := self.GetSAMLProvider()
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProvider")
	}
	for {
		_, err := provider.GetICloudroleByName(self.Name)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				break
			}
			return nil, errors.Wrapf(err, "GetICloudroleByName(%s)", self.Name)
		}
		info := strings.Split(self.Name, "-")
		num, err := strconv.Atoi(info[len(info)-1])
		if err != nil {
			info = append(info, "1")
		} else {
			info[len(info)-1] = fmt.Sprintf("%d", num+1)
		}
		self.Name = strings.Join(info, "-")
	}
	opts := &cloudprovider.SRoleCreateOptions{
		Name:         self.Name,
		Desc:         self.Description,
		SAMLProvider: sp.ExternalId,
	}
	iRole, err := provider.CreateICloudrole(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateICloudrole")
	}
	db.Update(self, func() error {
		self.ExternalId = iRole.GetGlobalId()
		self.Name = iRole.GetName()
		self.Document = iRole.GetDocument()
		self.Status = api.CLOUD_ROLE_STATUS_AVAILABLE
		return nil
	})
	return iRole, nil
}

func (self *SCloudrole) GetCloudpolicies() ([]SCloudpolicy, error) {
	q := CloudpolicyManager.Query()
	var sq *sqlchemy.SSubQuery
	if len(self.OwnerId) > 0 {
		su := SamluserManager.Query("cloudgroup_id").Equals("owner_id", self.OwnerId).Equals("cloudaccount_id", self.CloudaccountId).SubQuery()
		sq = CloudgroupPolicyManager.Query("cloudpolicy_id").In("cloudgroup_id", su).SubQuery()
	} else if len(self.CloudgroupId) > 0 {
		sq = CloudgroupPolicyManager.Query("cloudpolicy_id").Equals("cloudgroup_id", self.CloudgroupId).SubQuery()
	} else {
		return nil, fmt.Errorf("empty owner id or cloudgroup id")
	}
	q = q.In("id", sq)
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudrole) SyncRoles() error {
	iRole, err := self.GetICloudrole()
	if err != nil {
		return errors.Wrapf(err, "GetICloudrole")
	}
	policies, err := self.GetCloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetICloudpolicies")
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		if err != nil {
			return errors.Wrapf(err, "GetCloudaccount")
		}
	}
	cloudEnv := computeapi.GetCloudEnv(account.Provider, account.AccessUrl)
	local := set.New(set.ThreadSafe)
	for i := range policies {
		envs := strings.Split(policies[i].CloudEnv, ",")
		if !utils.IsInStringArray(cloudEnv, envs) {
			continue
		}
		if policies[i].PolicyType == api.CLOUD_POLICY_TYPE_SYSTEM {
			local.Add(policies[i].ExternalId)
		} else {
		}
	}
	iPolicies, err := iRole.GetICloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetICloudpolicies")
	}
	remote := set.New(set.ThreadSafe)
	for i := range iPolicies {
		remote.Add(iPolicies[i].GetGlobalId())
	}
	for _, id := range set.Difference(remote, local).List() {
		err = iRole.DetachPolicy(id.(string))
		if err != nil {
			return errors.Wrapf(err, "DetachPolicy(%s)", id)
		}
	}
	for _, id := range set.Difference(local, remote).List() {
		err = iRole.AttachPolicy(id.(string))
		if err != nil {
			return errors.Wrapf(err, "AttachPolicy(%s)", id)
		}
	}

	return nil
}

func (self *SCloudrole) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SCloudrole) syncWithCloudrole(ctx context.Context, userCred mcclient.TokenCredential, iRole cloudprovider.ICloudrole) error {
	_, err := db.Update(self, func() error {
		self.Name = self.GetName()
		self.Document = iRole.GetDocument()
		self.Status = api.CLOUD_ROLE_STATUS_AVAILABLE
		return nil
	})
	return err
}
