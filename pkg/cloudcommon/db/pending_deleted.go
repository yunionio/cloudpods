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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SPendingDeletedBaseManager struct{}

type SPendingDeletedBase struct {
	// 资源放入回收站时间
	PendingDeletedAt time.Time `json:"pending_deleted_at" list:"user" update:"admin"`
	// 资源是否处于回收站中
	PendingDeleted bool `nullable:"false" default:"false" index:"true" get:"user" list:"user" json:"pending_deleted"`
}

// GetPendingDeleted implements IPendingDeltable
func (base *SPendingDeletedBase) GetPendingDeleted() bool {
	return base.PendingDeleted
}

func (base *SPendingDeletedBase) MarkPendingDeleted() {
	base.PendingDeleted = true
	base.PendingDeletedAt = timeutils.UtcNow()
}

func (base *SPendingDeletedBase) CancelPendingDeleted() {
	base.PendingDeleted = false
	base.PendingDeletedAt = time.Time{}
}

// GetPendingDeletedAt implements IPendingDeltable
func (base *SPendingDeletedBase) GetPendingDeletedAt() time.Time {
	return base.PendingDeletedAt
}

func (base *SPendingDeletedBaseManager) FilterBySystemAttributes(manager IStandaloneModelManager, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	var pendingDelete string
	if query != nil {
		pendingDelete, _ = query.GetString("pending_delete")
	}
	pendingDeleteLower := strings.ToLower(pendingDelete)
	if pendingDeleteLower == "all" || pendingDeleteLower == "any" || utils.ToBool(pendingDeleteLower) {
		var isAllow bool
		allowScope, result := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "pending_delete")
		if result.Result.IsAllow() && !scope.HigherThan(allowScope) {
			isAllow = true
		}
		if !isAllow {
			pendingDeleteLower = ""
		}
	}

	if pendingDeleteLower == "all" || pendingDeleteLower == "any" {
	} else if utils.ToBool(pendingDeleteLower) {
		q = q.IsTrue("pending_deleted")
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
	}
	return q
}

func (base *SPendingDeletedBase) MarkPendingDelete(model IStandaloneModel, ctx context.Context, userCred mcclient.TokenCredential, newName string) error {
	if !base.PendingDeleted {
		_, err := Update(model, func() error {
			if len(newName) > 0 {
				model.SetName(newName)
			}
			model.MarkPendingDeleted()
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "MarkPendingDelete.Update")
		}
		OpsLog.LogEvent(model, ACT_PENDING_DELETE, model.GetShortDesc(ctx), userCred)
		logclient.AddSimpleActionLog(model, logclient.ACT_PENDING_DELETE, model.GetShortDesc(ctx), userCred, true)
	}
	return nil
}

func (base *SPendingDeletedBase) MarkCancelPendingDelete(model IStandaloneModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	manager := model.GetModelManager()
	ownerId := model.GetOwnerId()

	lockman.LockRawObject(ctx, manager.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

	newName, err := GenerateName(ctx, manager, ownerId, model.GetName())
	if err != nil {
		return errors.Wrapf(err, "GenerateNam")
	}
	_, err = Update(model, func() error {
		model.SetName(newName)
		model.CancelPendingDeleted()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "MarkCancelPendingDelete.Update")
	}
	OpsLog.LogEvent(model, ACT_CANCEL_DELETE, model.GetShortDesc(ctx), userCred)
	logclient.AddSimpleActionLog(model, logclient.ACT_CANCEL_DELETE, model.GetShortDesc(ctx), userCred, true)
	return nil
}
