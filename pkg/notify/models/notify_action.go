// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http//www.apache.org/licenses/LICENSE-2.0
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
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNotifyActionManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var NotifyActionManager *SNotifyActionManager

func init() {
	NotifyActionManager = &SNotifyActionManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SNotifyAction{},
			"notify_actions_tbl",
			"notify_action",
			"notify_actions",
		),
	}
	NotifyActionManager.SetVirtualObject(NotifyActionManager)
}

type SNotifyAction struct {
	db.SEnabledStatusStandaloneResourceBase
}

func (sm *SNotifyActionManager) InitializeData() error {
	ctx := context.Background()
	actions := []api.SAction{
		api.ActionCreate,
		api.ActionDelete,
		api.ActionPendingDelete,
		api.ActionUpdate,
		api.ActionRebuildRoot,
		api.ActionResetPassword,
		api.ActionChangeConfig,
		api.ActionExpiredRelease,
		api.ActionExecute,
		api.ActionChangeIpaddr,
		api.ActionSyncStatus,
		api.ActionCleanData,
		api.ActionMigrate,
		api.ActionCreateBackupServer,
		api.ActionDelBackupServer,
		api.ActionSyncCreate,
		api.ActionSyncUpdate,
		api.ActionSyncDelete,
		api.ActionOffline,
		api.ActionSystemPanic,
		api.ActionSystemException,
		api.ActionChecksumTest,
		api.ActionLock,
		api.ActionExceedCount,
		api.ActionSyncAccountStatus,
		api.ActionPasswordExpireSoon,
		api.ActionNetOutOfSync,
		api.ActionMysqlOutOfSync,
		api.ActionServiceAbnormal,
		api.ActionServerPanicked,
		api.ActionAttach,
		api.ActionDetach,
		api.ActionIsolatedDeviceCreate,
		api.ActionIsolatedDeviceUpdate,
		api.ActionIsolatedDeviceDelete,
		api.ActionStatusChanged,
		api.ActionStart,
		api.ActionStop,
		api.ActionRestart,
		api.ActionReset,
	}
	dbActions := []SNotifyAction{}
	q := NotifyActionManager.Query().In("id", actions)
	err := db.FetchModelObjects(NotifyActionManager, q, &dbActions)
	if err != nil {
		return errors.Wrap(err, "fetch topic_actions")
	}
	dbActionMap := map[api.SAction]struct{}{}
	for _, dbAction := range dbActions {
		dbActionMap[api.SAction(dbAction.Id)] = struct{}{}
	}
	for i, action := range actions {
		if _, ok := dbActionMap[action]; !ok {
			NotifyAction := SNotifyAction{}
			NotifyAction.Id = string(actions[i])
			NotifyAction.Name = string(actions[i])
			NotifyAction.Enabled = tristate.True
			NotifyActionManager.TableSpec().InsertOrUpdate(ctx, &NotifyAction)
		}
	}
	return nil
}

func (manager *SNotifyActionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SNotifyElementCreateInput) (api.SNotifyElementCreateInput, error) {
	_, err := manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validateCreate")
	}
	count, err := manager.Query().Equals("name", input.Name).CountWithError()
	if err != nil {
		return input, errors.Wrap(err, "fetch count")
	}
	if count > 0 {
		return input, errors.Wrap(httperrors.ErrDuplicateName, "%s has been exist")
	}
	return input, nil
}
