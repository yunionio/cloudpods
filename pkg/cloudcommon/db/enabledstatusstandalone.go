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

package db

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SEnabledStatusStandaloneResourceBase struct {
	SStatusStandaloneResourceBase

	Enabled bool `nullable:"false" default:"false" list:"user" create:"optional"` // = Column(Boolean, nullable=False, default=False)
}

type SEnabledStatusStandaloneResourceBaseManager struct {
	SStatusStandaloneResourceBaseManager
}

func NewEnabledStatusStandaloneResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SEnabledStatusStandaloneResourceBaseManager {
	return SEnabledStatusStandaloneResourceBaseManager{SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

type IEnableModel interface {
	IModel
	IsEnable() bool
	SetEnable() error
	SetDisable() error
}

func (self *SEnabledStatusStandaloneResourceBase) IsEnable() bool {
	return self.Enabled
}

func (self *SEnabledStatusStandaloneResourceBase) SetEnable() error {
	self.Enabled = true
	return nil
}

func (self *SEnabledStatusStandaloneResourceBase) SetDisable() error {
	self.Enabled = false
	return nil
}

func (self *SEnabledStatusStandaloneResourceBase) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return AllowPerformEnable(self, rbacutils.ScopeSystem, userCred)
}

func AllowPerformEnable(obj IEnableModel, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential) bool {
	return IsAllowPerform(scope, userCred, obj, "enable")
}

func (self *SEnabledStatusStandaloneResourceBase) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformEnable(self, userCred)
}

func PerformEnable(obj IEnableModel, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	if !obj.IsEnable() {
		_, err := Update(obj, func() error {
			if err := obj.SetEnable(); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Errorf("PerformEnable save update fail %s", err)
			return nil, err
		}
		OpsLog.LogEvent(obj, ACT_ENABLE, "", userCred)
		logclient.AddSimpleActionLog(obj, logclient.ACT_ENABLE, nil, userCred, true)
	}
	return nil, nil
}

func (self *SEnabledStatusStandaloneResourceBase) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return AllowPerformDisable(self, rbacutils.ScopeSystem, userCred)
}

func AllowPerformDisable(obj IEnableModel, scope rbacutils.TRbacScope, userCred mcclient.TokenCredential) bool {
	return IsAllowPerform(scope, userCred, obj, "disable")
}

func (self *SEnabledStatusStandaloneResourceBase) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformDisable(self, userCred)
}

func PerformDisable(obj IEnableModel, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	if obj.IsEnable() {
		_, err := Update(obj, func() error {
			if err := obj.SetDisable(); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Errorf("PerformDisable save update fail %s", err)
			return nil, err
		}
		OpsLog.LogEvent(obj, ACT_DISABLE, "", userCred)
		logclient.AddSimpleActionLog(obj, logclient.ACT_DISABLE, nil, userCred, true)
	}
	return nil, nil
}

func (manager *SEnabledStatusStandaloneResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.EnabledStatusStandaloneResourceCreateInput) (apis.EnabledStatusStandaloneResourceCreateInput, error) {
	var err error
	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SEnabledStatusStandaloneResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.EnabledStatusStandaloneResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	return ListEnableItemFilter(q, query.Enabled)
}

func ListEnableItemFilter(q *sqlchemy.SQuery, enabled *bool) (*sqlchemy.SQuery, error) {
	if enabled != nil {
		if *enabled {
			q = q.IsTrue("enabled")
		} else {
			q = q.IsFalse("enabled")
		}
	}
	return q, nil
}
