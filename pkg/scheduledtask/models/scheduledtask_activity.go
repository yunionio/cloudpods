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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/scheduledtask"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var ScheduledTaskActivityManager *SScheduledTaskActivityManager

func init() {
	ScheduledTaskActivityManager = &SScheduledTaskActivityManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SScheduledTaskActivity{},
			"scheduledtaskactivities_tbl",
			"scheduledtaskactivity",
			"scheduledtaskactivities",
		),
	}
	ScheduledTaskActivityManager.SetVirtualObject(ScheduledTaskActivityManager)
}

// +onecloud:swagger-gen-model-singular=scheduledtaskactivity
// +onecloud:swagger-gen-model-singular=scheduledtaskactivities
type SScheduledTaskActivityManager struct {
	db.SStatusStandaloneResourceBaseManager
}

type SScheduledTaskActivity struct {
	db.SStatusStandaloneResourceBase
	ScheduledTaskId string    `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true"`
	StartTime       time.Time `list:"user"`
	EndTime         time.Time `list:"user"`
	Reason          string    `charset:"utf8" list:"user"`
}

func (sam *SScheduledTaskActivityManager) InitializeData() error {
	sas := make([]SScheduledTaskActivity, 0, 10)
	q := ScheduledTaskActivityManager.Query().Equals("status", api.ST_ACTIVITY_STATUS_EXEC)
	err := db.FetchModelObjects(ScheduledTaskActivityManager, q, &sas)
	if err != nil {
		return err
	}
	for i := range sas {
		err := sas[i].Fail("As the service restarts, the status becomes unknown")
		if err != nil {
			return err
		}
	}
	return nil
}

func (sam *SScheduledTaskActivityManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	return sam.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
}

func (sam *SScheduledTaskActivityManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ScheduledTaskActivityListInput) (*sqlchemy.SQuery, error) {
	return sam.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
}

func (sam *SScheduledTaskActivityManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ScheduledTaskActivityDetails {
	rows := make([]api.ScheduledTaskActivityDetails, len(objs))
	statusRows := sam.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].StatusStandaloneResourceDetails = statusRows[i]
	}
	return rows
}

func (sam *SScheduledTaskActivityManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.ScheduledTaskActivityListInput) (*sqlchemy.SQuery, error) {
	q, err := sam.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q = q.Desc("start_time").Desc("end_time")
	if len(input.ScheduledTask) == 0 {
		return nil, httperrors.NewInputParameterError("need scheduled task")
	}
	task, err := ScheduledTaskManager.FetchByIdOrName(ctx, userCred, input.ScheduledTask)
	if err != nil {
		return nil, err
	}
	q = q.Equals("scheduled_task_id", task.GetId())
	return q, nil
}

func (sam *SScheduledTaskActivityManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (sam *SScheduledTaskActivityManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (sam *SScheduledTaskActivityManager) FileterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject, rbacscope.ScopeDomain:
			scheduledTaskQ := ScheduledTaskManager.Query("id", "domain_id").SubQuery()
			q = q.Join(scheduledTaskQ, sqlchemy.Equals(q.Field("scheduled_task_id"), scheduledTaskQ.Field("id")))
			q = q.Filter(sqlchemy.Equals(scheduledTaskQ.Field("domain_id"), owner.GetProjectDomainId()))
		}
	}
	return q
}

func (sam *SScheduledTaskActivityManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchDomainInfo(ctx, data)
}

func (sa *SScheduledTaskActivity) GetOwnerId() mcclient.IIdentityProvider {
	obj, _ := ScheduledTaskManager.FetchById(sa.ScheduledTaskId)
	if obj == nil {
		return nil
	}
	return obj.GetOwnerId()
}

func (sa *SScheduledTaskActivity) SetResult(status, reason string) error {
	_, err := db.Update(sa, func() error {
		sa.Status = status
		sa.Reason = reason
		sa.EndTime = time.Now()
		return nil
	})
	return err
}

func (sa *SScheduledTaskActivity) Fail(reason string) error {
	return sa.SetResult(api.ST_ACTIVITY_STATUS_FAILED, reason)
}

func (sa *SScheduledTaskActivity) Succeed() error {
	return sa.SetResult(api.ST_ACTIVITY_STATUS_SUCCEED, "")
}

func (sa *SScheduledTaskActivity) PartFail(reason string) error {
	return sa.SetResult(api.ST_ACTIVITY_STATUS_PART_SUCCEED, reason)
}
