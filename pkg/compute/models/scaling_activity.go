package models

import (
	"context"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/errors"
)

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
	Reason     string    `width:"256" charset:"ascii" get:"user" list:"user"`
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

func (sam *SScalingActivity) SetResult(action string, success bool, failReason string) error {
	_, err := db.Update(sam, func() error {
		if len(action) != 0 {
			sam.ActionDesc = action
		}
		sam.EndTime = time.Now()
		if success {
			sam.Status = compute.SA_STATUS_SUCCEED
		} else {
			sam.Status = compute.SA_STATUS_FAILED
			sam.Reason = failReason
		}
		return nil
	})
	return err
}

func (sam *SScalingActivityManager) CreateScalingActivity(sgId, triggerDesc, status string) (*SScalingActivity, error) {
	scalingActivity := &SScalingActivity{
		TriggerDesc: triggerDesc,
		StartTime:   time.Now(),
	}
	scalingActivity.ScalingGroupId = sgId
	scalingActivity.Status = status
	scalingActivity.SetModelManager(sam, scalingActivity)
	return scalingActivity, sam.TableSpec().Insert(scalingActivity)
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

func (sam *SScalingActivity) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, isList bool) (compute.ScalingActivityDetails, error) {
	return compute.ScalingActivityDetails{}, nil
}

func (sam *SScalingActivityManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, input compute.ScalingActivityListInput) (*sqlchemy.SQuery, error) {

	q, err := sam.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	return sam.SScalingGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ScalingGroupFilterListInput)
}
