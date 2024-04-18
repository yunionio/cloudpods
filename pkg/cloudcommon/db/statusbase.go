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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
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

func StatusBaseSetStatus(ctx context.Context, model IStatusBaseModel, userCred mcclient.TokenCredential, status string, reason string) error {
	return statusBaseSetStatus(ctx, model, userCred, status, reason)
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

func statusBaseSetStatus(ctx context.Context, model IStatusBaseModel, userCred mcclient.TokenCredential, status string, reason string) error {
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
	CallStatusChanegdNotifyHook(ctx, userCred, oldStatus, status, model)
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		OpsLog.LogEvent(model, ACT_UPDATE_STATUS, notes, userCred)
		success := true
		if strings.Contains(status, "fail") || status == apis.STATUS_UNKNOWN || status == api.CLOUD_PROVIDER_DISCONNECTED {
			success = false
		}
		logclient.AddSimpleActionLog(model, logclient.ACT_UPDATE_STATUS, notes, userCred, success)
	}
	return nil
}

func StatusBasePerformStatus(ctx context.Context, model IStatusBaseModel, userCred mcclient.TokenCredential, input apis.PerformStatusInput) error {
	if len(input.Status) == 0 {
		return httperrors.NewMissingParameterError("status")
	}
	err := statusBaseSetStatus(ctx, model, userCred, input.Status, input.Reason)
	if err != nil {
		return errors.Wrap(err, "statusBaseSetStatus")
	}
	return nil
}

func (model *SStatusResourceBase) IsInStatus(status ...string) bool {
	return utils.IsInArray(model.Status, status)
}

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
