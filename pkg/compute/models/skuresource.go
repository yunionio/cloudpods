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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func PerformActionSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, resourceKey string, input apis.SkuSyncInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(resourceKey, []string{
		ServerSkuManager.Keyword(),
		ElasticcacheSkuManager.Keyword(),
		DBInstanceSkuManager.Keyword(),
		NatSkuManager.Keyword(),
		NasSkuManager.Keyword(),
	}) {
		return nil, httperrors.NewUnsupportOperationError("resource %s is not support sync skus", resourceKey)
	}

	if len(input.Provider) == 0 && len(input.CloudregionIds) == 0 {
		return nil, httperrors.NewMissingParameterError("must specific one of `provider` or `cloudregionids`")
	}

	input.Provider = strings.TrimSpace(input.Provider)
	if len(input.Provider) > 0 {
		if !utils.IsInStringArray(input.Provider, cloudprovider.GetPublicProviders()) {
			return nil, httperrors.NewInputParameterError("Unsupported provider %s", input.Provider)
		}
	}

	// cloudregions to sync
	q := CloudregionManager.Query()
	if len(input.Provider) > 0 {
		q = q.Equals("provider", input.Provider)
	} else {
		q = q.In("provider", cloudprovider.GetPublicProviders())
	}

	if len(input.CloudregionIds) > 0 {
		q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("id"), input.CloudregionIds), sqlchemy.Equals(q.Field("name"), input.CloudregionIds)))
	}

	regions := []SCloudregion{}
	err := db.FetchModelObjects(CloudregionManager, q, &regions)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(input.CloudregionIds) > 0 && len(input.CloudregionIds) != len(regions) {
		return nil, httperrors.NewInputParameterError("input data contains invalid cloudregion id")
	}

	if len(regions) == 0 {
		return nil, httperrors.NewInputParameterError("no cloudregion found to sync skus")
	}

	params := jsonutils.NewDict()
	params.Set("resource", jsonutils.NewString(resourceKey))
	ret := jsonutils.NewDict()
	taskIds := jsonutils.NewArray()
	for i := range regions {
		err = regions[i].StartSyncSkusTask(ctx, userCred, resourceKey)
		if err != nil {
			return nil, err
		}
	}
	ret.Set("tasks", taskIds)
	return ret, nil
}

func GetPropertySkusSyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query apis.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	tasks := []taskman.STask{}
	q := taskman.TaskManager.Query()
	q = q.Equals("obj_name", CloudregionManager.Keyword())
	q = q.Equals("task_name", "CloudRegionSyncSkusTask")
	if len(query.TaskIds) > 0 {
		q = q.In("id", query.TaskIds)
	} else {
		q = q.NotIn("stage", []string{taskman.TASK_STAGE_FAILED, taskman.TASK_STAGE_COMPLETE})
	}
	err := q.All(&tasks)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	ret := jsonutils.NewDict()
	items := jsonutils.NewArray()
	for i := range tasks {
		item := jsonutils.NewDict()
		item.Set("id", jsonutils.NewString(tasks[i].GetId()))
		item.Set("created_at", jsonutils.NewTimeString(tasks[i].GetStartTime()))
		item.Set("stage", jsonutils.NewString(tasks[i].Stage))
		items.Add(item)
	}
	ret.Set("tasks", items)
	return ret, nil
}
