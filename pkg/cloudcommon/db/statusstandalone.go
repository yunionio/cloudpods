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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStatusStandaloneResourceBase struct {
	SStandaloneResourceBase
	SStatusResourceBase
}

type SStatusStandaloneResourceBaseManager struct {
	SStandaloneResourceBaseManager
	SStatusResourceBaseManager
}

func NewStatusStandaloneResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStatusStandaloneResourceBaseManager {
	return SStatusStandaloneResourceBaseManager{
		SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (self *SStatusStandaloneResourceBase) GetIStatusStandaloneModel() IStatusStandaloneModel {
	return self.GetVirtualObject().(IStatusStandaloneModel)
}

func (manager *SStatusStandaloneResourceBaseManager) GetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*apis.StatusStatistic, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}

	var err error
	q := manager.Query()
	q, err = ListItemQueryFilters(im, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	sq := q.SubQuery()
	statQ := sq.Query(sq.Field("status"), sqlchemy.COUNT("total_count", sq.Field("id")))
	statQ = statQ.GroupBy(sq.Field("status"))

	ret := []struct {
		Status     string
		TotalCount int64
	}{}
	err = statQ.All(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}
	result := &apis.StatusStatistic{
		StatusInfo: []apis.StatusStatisticStatusInfo{},
	}
	for _, s := range ret {
		result.StatusInfo = append(result.StatusInfo, apis.StatusStatisticStatusInfo{
			Status:     s.Status,
			TotalCount: s.TotalCount,
		})
	}
	return result, nil
}

// 更新资源状态
func (self *SStatusStandaloneResourceBase) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	err := StatusBasePerformStatus(self.GetIStatusStandaloneModel(), userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "StatusBasePerformStatus")
	}
	return nil, nil
}

func (model *SStatusStandaloneResourceBase) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	return statusBaseSetStatus(model.GetIStatusStandaloneModel(), userCred, status, reason)
}

func (model *SStatusStandaloneResourceBase) SetProgress(progress float32) error {
	return statusBaseSetProgress(model.GetIStatusStandaloneModel(), progress)
}

func (manager *SStatusStandaloneResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.StatusStandaloneResourceCreateInput) (apis.StatusStandaloneResourceCreateInput, error) {
	var err error
	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SStatusStandaloneResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusStandaloneResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SStatusStandaloneResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.StatusStandaloneResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SStatusResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SStatusStandaloneResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStatusStandaloneResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StatusStandaloneResourceDetails {
	rows := make([]apis.StatusStandaloneResourceDetails, len(objs))
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.StatusStandaloneResourceDetails{
			StandaloneResourceDetails: stdRows[i],
		}
	}
	return rows
}

func (model *SStatusStandaloneResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.StatusStandaloneResourceBaseUpdateInput,
) (apis.StatusStandaloneResourceBaseUpdateInput, error) {
	var err error
	input.StandaloneResourceBaseUpdateInput, err = model.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}
