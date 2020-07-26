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

package taskman

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type ITask interface {
	cloudcommon.IStartable

	ScheduleRun(data jsonutils.JSONObject)
	GetParams() *jsonutils.JSONDict
	GetUserCred() mcclient.TokenCredential
	GetTaskId() string
	SetStage(stageName string, data *jsonutils.JSONDict) error

	GetTaskRequestHeader() http.Header

	SetStageComplete(ctx context.Context, data *jsonutils.JSONDict)
	SetStageFailed(ctx context.Context, reason jsonutils.JSONObject)

	GetPendingUsage(quota quotas.IQuota, index int) error
	ClearPendingUsage(index int) error
}
