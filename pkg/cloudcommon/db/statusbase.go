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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SStatusResourceBaseManager struct{}

type SStatusResourceBase struct {
	// 资源状态
	Status string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user" create:"optional" json:"status"`

	// 操作进度0-100
	Progress float32 `list:"user" update:"user" default:"100" json:"progress" log:"skip"`
}

type IStatusBase interface {
	SetStatusValue(status string)
	GetStatus() string
	SetProgressValue(progress float32)
	GetProgress() float32
}

type IStatusBaseModel interface {
	IModel
	IStatusBase
}

func (model *SStatusResourceBase) SetStatusValue(status string) {
	model.Status = status
	model.Progress = 0
}

func (model *SStatusResourceBase) SetProgressValue(progress float32) {
	model.Progress = progress
}

func (model SStatusResourceBase) GetStatus() string {
	return model.Status
}

func (model SStatusResourceBase) GetProgress() float32 {
	return model.Progress
}

func StatusBaseSetStatus(model IStatusBaseModel, userCred mcclient.TokenCredential, status string, reason string) error {
	return statusBaseSetStatus(model, userCred, status, reason)
}

func statusBaseSetProgress(model IStatusBaseModel, progress float32) error {
	_, err := Update(model, func() error {
		model.SetProgressValue(progress)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func statusBaseSetStatus(model IStatusBaseModel, userCred mcclient.TokenCredential, status string, reason string) error {
	if model.GetStatus() == status {
		return nil
	}
	oldStatus := model.GetStatus()
	_, err := Update(model, func() error {
		model.SetStatusValue(status)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		OpsLog.LogEvent(model, ACT_UPDATE_STATUS, notes, userCred)
		logclient.AddSimpleActionLog(model, logclient.ACT_UPDATE_STATUS, notes, userCred, true)
	}
	return nil
}

func StatusBasePerformStatus(model IStatusBaseModel, userCred mcclient.TokenCredential, input apis.PerformStatusInput) error {
	if len(input.Status) == 0 {
		return httperrors.NewMissingParameterError("status")
	}
	err := statusBaseSetStatus(model, userCred, input.Status, input.Reason)
	if err != nil {
		return errors.Wrap(err, "statusBaseSetStatus")
	}
	return nil
}

func (model *SStatusResourceBase) IsInStatus(status ...string) bool {
	return utils.IsInStringArray(model.Status, status)
}

/*func (model *SStatusStandaloneResourceBase) AllowGetDetailsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAllowGetSpec(rbacutils.ScopeSystem, userCred, model, "status")
}*/

// 获取资源状态
func (model *SStatusResourceBase) GetDetailsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (apis.GetDetailsStatusOutput, error) {
	ret := apis.GetDetailsStatusOutput{}
	ret.Status = model.Status
	return ret, nil
}

func (manager *SStatusResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.Status) > 0 {
		q = q.In("status", query.Status)
	}
	return q, nil
}

func (manager *SStatusResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	return q, nil
}
