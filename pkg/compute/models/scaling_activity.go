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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=scalingactivity
// +onecloud:swagger-gen-model-plural=scalingactivities
type SScalingActivityManager struct {
	db.SStatusStandaloneResourceBaseManager
	SScalingGroupResourceBaseManager
}

type SScalingActivity struct {
	db.SStatusStandaloneResourceBase

	SScalingGroupResourceBase
	InstanceNumber int `list:"user" get:"user" default:"-1"`
	// 起因描述
	TriggerDesc string `width:"256" charset:"ascii" get:"user" list:"user"`
	// 行为描述
	ActionDesc string    `width:"256" charset:"ascii" get:"user" list:"user"`
	StartTime  time.Time `list:"user" get:"user"`
	EndTime    time.Time `list:"user" get:"user"`
	Reason     string    `width:"1024" charset:"ascii" get:"user" list:"user"`
}

var ScalingActivityManager *SScalingActivityManager

func init() {
	ScalingActivityManager = &SScalingActivityManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SScalingActivity{},
			"scalingactivities_tbl",
			"scalingactivity",
			"scalingactivities",
		),
	}
	ScalingActivityManager.SetVirtualObject(ScalingActivityManager)
}

func (sam *SScalingActivityManager) FetchByStatus(ctx context.Context, saIds, status []string, action string) (ids []string, err error) {
	q := sam.Query("id").In("id", saIds)
	if action == "not" {
		q = q.NotIn("status", status)
	} else {
		q = q.In("status", status)
	}
	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "sQuery.Rows")
	}
	defer rows.Close()
	var id string
	for rows.Next() {
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return
}

func (sam *SScalingActivity) SetFailed(actionDesc, reason string) error {
	return sam.SetResult(actionDesc, compute.SA_STATUS_FAILED, reason, -1)
}

func (sam *SScalingActivity) SetResult(actionDesc, status, reason string, instanceNum int) error {
	_, err := db.Update(sam, func() error {
		if len(actionDesc) != 0 {
			sam.ActionDesc = actionDesc
		}
		sam.EndTime = time.Now()
		sam.Status = status
		if len(reason) != 0 {
			sam.Reason = reason
		}
		if instanceNum != -1 {
			sam.InstanceNumber = instanceNum
		}
		return nil
	})
	return err
}

func (sam *SScalingActivity) SetReject(action string, reason string) error {
	return sam.SetResult(action, compute.SA_STATUS_REJECT, reason, -1)
}

func (sam *SScalingActivityManager) CreateScalingActivity(ctx context.Context, sgId, triggerDesc, status string) (*SScalingActivity, error) {
	scalingActivity := &SScalingActivity{
		TriggerDesc: triggerDesc,
		StartTime:   time.Now(),
	}
	scalingActivity.ScalingGroupId = sgId
	scalingActivity.Status = status
	scalingActivity.SetModelManager(sam, scalingActivity)
	return scalingActivity, sam.TableSpec().Insert(ctx, scalingActivity)
}

func (sa *SScalingActivity) StartToScale(triggerDesc string) (*SScalingActivity, error) {
	_, err := db.Update(sa, func() error {
		sa.TriggerDesc = triggerDesc
		sa.Status = compute.SA_STATUS_EXEC
		return nil
	})
	return sa, err
}

func (sa *SScalingActivity) SimpleDelete() error {
	_, err := db.Update(sa, func() error {
		sa.MarkDelete()
		return nil
	})
	return err
}

func (sam *SScalingActivityManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := sam.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return sam.SScalingGroupResourceBaseManager.QueryDistinctExtraField(q, field)
}

func (sam *SScalingActivityManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query compute.ScalingActivityListInput) (*sqlchemy.SQuery, error) {
	return sam.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
}

func (sam *SScalingActivityManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []compute.ScalingActivityDetails {
	rows := make([]compute.ScalingActivityDetails, len(objs))
	statusRows := sam.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	sgRows := sam.SScalingGroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].StatusStandaloneResourceDetails = statusRows[i]
		rows[i].ScalingGroupResourceInfo = sgRows[i]
	}
	return rows
}

func (sam *SScalingActivityManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, input compute.ScalingActivityListInput) (*sqlchemy.SQuery, error) {

	q, err := sam.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = sam.SScalingGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ScalingGroupFilterListInput)
	if err != nil {
		return nil, err
	}
	q = q.Desc("start_time").Desc("end_time")
	return q, nil
}

func (sam *SScalingActivityManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (sam *SScalingActivityManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (sam *SScalingActivityManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject, rbacscope.ScopeDomain:
			scalingGroupQ := ScalingGroupManager.Query("id", "domain_id").SubQuery()
			q = q.Join(scalingGroupQ, sqlchemy.Equals(q.Field("scaling_group_id"), scalingGroupQ.Field("id")))
			q = q.Filter(sqlchemy.Equals(scalingGroupQ.Field("domain_id"), owner.GetProjectDomainId()))
		}
	}
	return q
}

func (sam *SScalingActivityManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchDomainInfo(ctx, data)
}

func (sa *SScalingActivity) GetOwnerId() mcclient.IIdentityProvider {
	scalingGroup := sa.GetScalingGroup()
	if scalingGroup != nil {
		return scalingGroup.GetOwnerId()
	}
	return nil
}
