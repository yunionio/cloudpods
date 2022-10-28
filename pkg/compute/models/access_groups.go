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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAccessGroupManager struct {
	db.SStatusInfrasResourceBaseManager
}

var AccessGroupManager *SAccessGroupManager

func init() {
	AccessGroupManager = &SAccessGroupManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SAccessGroup{},
			"access_groups_tbl",
			"access_group",
			"access_groups",
		),
	}
	AccessGroupManager.SetVirtualObject(AccessGroupManager)
}

type SAccessGroup struct {
	db.SStatusInfrasResourceBase

	IsDirty bool `nullable:"false" default:"false"`
}

func (manager *SAccessGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager SAccessGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AccessGroupDetails {
	rows := make([]api.AccessGroupDetails, len(objs))
	stdRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.AccessGroupDetails{
			StatusInfrasResourceBaseDetails: stdRows[i],
		}
	}
	return rows
}

func (manager *SAccessGroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SAccessGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, nil
}

func (manager *SAccessGroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (self *SAccessGroup) GetChangeOwnerCandidateDomainIds() []string {
	return []string{}
}

func (self *SAccessGroup) GetEmptyAccessGroupRuleInfo() cloudprovider.AccessGroupRuleInfo {
	return cloudprovider.AccessGroupRuleInfo{
		MinPriority: 1,
		MaxPriority: 100,
		Rules:       cloudprovider.AccessGroupRuleSet{},
		SupportedUserAccessType: []cloudprovider.TUserAccessType{
			cloudprovider.UserAccessTypeRootSquash,
			cloudprovider.UserAccessTypeNoRootSquash,
			cloudprovider.UserAccessTypeAllSquash,
		},
	}
}

func (self *SAccessGroup) GetAccessGroupRuleInfo() (cloudprovider.AccessGroupRuleInfo, error) {
	ret := self.GetEmptyAccessGroupRuleInfo()
	rules, err := self.GetAccessGroupRules()
	if err != nil {
		return ret, errors.Wrapf(err, "GetAccessGroupRules")
	}
	for _, rule := range rules {
		userType := cloudprovider.TUserAccessType(rule.UserAccessType)
		if isIn, _ := utils.InArray(userType, ret.SupportedUserAccessType); !isIn {
			ret.SupportedUserAccessType = append(ret.SupportedUserAccessType, userType)
		}
		ret.Rules = append(ret.Rules, cloudprovider.AccessGroupRule{
			ExternalId:     rule.Id,
			Source:         rule.Source,
			RWAccessType:   cloudprovider.TRWAccessType(rule.RWAccessType),
			UserAccessType: userType,
			Priority:       rule.Priority,
		})
	}
	return ret, nil
}

func (self *SAccessGroup) GetAccessGroupRules() ([]SAccessGroupRule, error) {
	rules := []SAccessGroupRule{}
	q := AccessGroupRuleManager.Query().Equals("access_group_id", self.Id)
	err := db.FetchModelObjects(AccessGroupRuleManager, q, &rules)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return rules, nil
}

func (manager *SAccessGroupManager) InitializeData() error {
	_, err := manager.FetchById(api.DEFAULT_ACCESS_GROUP)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrapf(err, "manager.FetchById(%s)", api.DEFAULT_ACCESS_GROUP)
		}
		log.Infof("Init default access group")
		accessGroup := &SAccessGroup{}
		accessGroup.SetModelManager(manager, accessGroup)
		accessGroup.Id = api.DEFAULT_ACCESS_GROUP
		accessGroup.Name = "Default"
		accessGroup.Status = api.ACCESS_GROUP_STATUS_AVAILABLE
		accessGroup.DomainId = auth.AdminCredential().GetProjectDomainId()
		accessGroup.IsPublic = true
		accessGroup.Deleted = false
		err = manager.TableSpec().InsertOrUpdate(context.TODO(), accessGroup)
		if err != nil {
			return errors.Wrapf(err, "InsertOrUpdate default access group")
		}
	}
	return nil
}

func (manager *SAccessGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.AccessGroupCreateInput) (api.AccessGroupCreateInput, error) {
	var err error
	input.StatusInfrasResourceBaseCreateInput, err = manager.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	input.Status = api.ACCESS_GROUP_STATUS_AVAILABLE
	return input, nil
}

func (self *SAccessGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SAccessGroup) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "AccessGroupDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.ACCESS_GROUP_STATUS_DELETE_FAILED, err.Error())
		return nil
	}
	self.SetStatus(userCred, api.ACCESS_GROUP_STATUS_DELETING, "")
	return nil
}

func (self *SAccessGroup) GetMountTargets() ([]SMountTarget, error) {
	mts := []SMountTarget{}
	q := MountTargetManager.Query().Equals("access_group_id", self.Id)
	err := db.FetchModelObjects(MountTargetManager, q, &mts)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return mts, nil
}

func (self *SAccessGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SAccessGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SAccessGroup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.Id == api.DEFAULT_ACCESS_GROUP {
		return httperrors.NewProtectedResourceError("not allow to delete default access group")
	}
	mts, err := self.GetMountTargets()
	if err != nil {
		return errors.Wrapf(err, "GetMountTargets")
	}
	if len(mts) > 0 {
		return httperrors.NewNotEmptyError("access group not empty, please delete mount target first")
	}
	return self.SStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SAccessGroup) DoSync(ctx context.Context, userCred mcclient.TokenCredential) {
	if _, err := db.Update(self, func() error {
		self.IsDirty = true
		return nil
	}); err != nil {
		log.Errorf("Update Security Group error: %s", err.Error())
	}
	time.AfterFunc(10*time.Second, func() {
		AccessGroupManager.DelaySync(context.Background(), userCred, self.Id)
	})
}

func (manager *SAccessGroupManager) DelaySync(ctx context.Context, userCred mcclient.TokenCredential, idStr string) error {
	_group, err := manager.FetchById(idStr)
	if err != nil {
		return errors.Wrapf(err, "FetchById(%s)", idStr)
	}
	group := _group.(*SAccessGroup)
	needSync := false

	func() {
		lockman.LockObject(ctx, group)
		defer lockman.ReleaseObject(ctx, group)

		if group.IsDirty {
			if _, err := db.Update(group, func() error {
				group.IsDirty = false
				return nil
			}); err != nil {
				log.Errorf("Update Access Group error: %s", err.Error())
			}
			needSync = true
		}
	}()

	if needSync {
		return group.StartSyncRulesTask(ctx, userCred, "")
	}
	return nil
}

func (self *SAccessGroup) StartSyncRulesTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "AccessGroupSyncRulesTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.ACCESS_GROUP_STATUS_SYNC_RULES_FAILED, err.Error())
		return err
	}
	self.SetStatus(userCred, api.ACCESS_GROUP_STATUS_SYNC_RULES, "")
	return nil
}

// 同步权限组状态
func (self *SAccessGroup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MountTargetSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SAccessGroup) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "AccessGroupSyncstatusTask", parentTaskId)
}
