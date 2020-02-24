package models

import (
	"context"
	"time"
	"yunion.io/x/onecloud/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/errors"
)

type SScalingActivityManager struct {
	db.SStatusStandaloneResourceBaseManager
}

type SScalingActivity struct {
	db.SStatusStandaloneResourceBase

	ScalingPolicyId string `width:"128" charset:"ascii"`
	// 起因描述
	TriggerDesc string `width:"256" charset:"ascii" get:"user" list:"user"`
	// 行为描述
	ActionDesc   string    `width:"256" charset:"ascii" get:"user" list:"user"`
	StartTime    time.Time `list:"user" get:"user"`
	EndTime      time.Time `list:"user" get:"user"`
	FailReason string    `width:"256" charset:"ascii" get:"user" list:"user"`
}

var ScalingActivityManager *SScalingActivityManager

func init() {
	ScalingActivityManager = &SScalingActivityManager{
		db.NewStatusStandaloneResourceBaseManager(
			SScalingActivity{},
			"scalingactivities_tbl",
			"scalingactivities",
			"scalingactivity",
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
	if err == nil {
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
			sam.FailReason = failReason
		}
		return nil
	})
	return err
}
