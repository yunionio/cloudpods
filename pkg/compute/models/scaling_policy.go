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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=scalingpolicy
// +onecloud:swagger-gen-model-plural=scalingpolicies
type SScalingPolicyManager struct {
	db.SVirtualResourceBaseManager
	SScalingGroupResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SScalingPolicy struct {
	db.SVirtualResourceBase
	SScalingGroupResourceBase
	db.SEnabledResourceBase

	TriggerType string `width:"16" charset:"ascii" default:"timing" create:"required" list:"user"`
	TriggerId   string `width:"128" charset:"ascii"`

	// Action of scaling activity
	Action string `width:"8" charset:"ascii" default:"set" create:"required" list:"user"`
	Number int    `nullable:"false" default:"1" create:"required" list:"user"`

	// Unit of Number
	Unit string `width:"4" charset:"ascii" create:"required" list:"user"`

	// Scaling activity triggered by alarms will be rejected during this period about CoolingTime
	CoolingTime int `nullable:"false" default:"300" create:"required" list:"user"`
}

var ScalingPolicyManager *SScalingPolicyManager

func init() {
	ScalingPolicyManager = &SScalingPolicyManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SScalingPolicy{},
			"scalingpolicies_tbl",
			"scalingpolicy",
			"scalingpolicies",
		),
	}
	ScalingPolicyManager.SetVirtualObject(ScalingPolicyManager)
}

func (spm *SScalingPolicyManager) ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential,
	query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var err error
	query, err = spm.SVirtualResourceBaseManager.ValidateListConditions(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	if !query.Contains("scaling_group") {
		return nil, httperrors.NewInputParameterError("every scaling policy belong to a scaling group")
	}
	return query, nil
}

func (spm *SScalingPolicyManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, input api.ScalingPolicyListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = spm.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return q, err
	}
	q, err = spm.SScalingGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ScalingGroupFilterListInput)
	if err != nil {
		return q, err
	}
	q, err = spm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return q, err
	}
	if len(input.TriggerType) != 0 {
		q = q.Equals("trigger_type", input.TriggerType)
	}
	return q, nil
}

func (spm *SScalingPolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := spm.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return spm.SScalingGroupResourceBaseManager.QueryDistinctExtraField(q, field)
}

func (sgm *SScalingPolicy) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{"scaling_group_id": sgm.ScalingGroupId})
}

func (spm *SScalingPolicyManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	return spm.SScalingGroupResourceBaseManager.FetchUniqValues(ctx, data)
}

func (spm *SScalingPolicyManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	return spm.SScalingGroupResourceBaseManager.FilterByUniqValues(q, values)
}

func (spm *SScalingPolicyManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query api.ScalingPolicyListInput) (*sqlchemy.SQuery, error) {
	return spm.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
}

func (spm *SScalingPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ScalingPolicyDetails {
	rows := make([]api.ScalingPolicyDetails, len(objs))
	statusRows := spm.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	sgRows := spm.SScalingGroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	var err error
	for i := range rows {
		rows[i], err = objs[i].(*SScalingPolicy).getMoreDetails(ctx, userCred, query, isList)
		if err != nil {
			log.Errorf("SScalingPolicy.getMoreDetails error: %s", err)
		}
		rows[i].VirtualResourceDetails = statusRows[i]
		rows[i].ScalingGroupResourceInfo = sgRows[i]
	}
	return rows
}

func (sp *SScalingPolicy) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, isList bool) (api.ScalingPolicyDetails, error) {

	var out api.ScalingPolicyDetails
	switch sp.TriggerType {
	case api.TRIGGER_ALARM:
		model, err := ScalingAlarmManager.FetchById(sp.TriggerId)
		if errors.Cause(err) == sql.ErrNoRows {
			return out, nil
		}
		if err != nil {
			return out, errors.Wrap(err, "ScalingAlarmManager.FetchById")
		}
		out.Alarm = model.(*SScalingAlarm).AlarmDetails()
	case api.TRIGGER_TIMING:
		model, err := ScalingTimerManager.FetchById(sp.TriggerId)
		if errors.Cause(err) == sql.ErrNoRows {
			return out, nil
		}
		if err != nil {
			return out, errors.Wrap(err, "ScalingTimerManager.FetchById")
		}
		out.Timer = model.(*SScalingTimer).TimerDetails()
	case api.TRIGGER_CYCLE:
		model, err := ScalingTimerManager.FetchById(sp.TriggerId)
		if errors.Cause(err) == sql.ErrNoRows {
			return out, nil
		}
		if err != nil {
			return out, errors.Wrap(err, "ScalingTimerManager.FetchById")
		}
		out.CycleTimer = model.(*SScalingTimer).CycleTimerDetails()
	}

	return out, nil
}

func (spm *SScalingPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ScalingPolicyCreateInput) (
	api.ScalingPolicyCreateInput, error) {
	log.Debugf("insert validateCreateData")
	var err error
	input.VirtualResourceCreateInput, err = spm.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query,
		input.VirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	// check scaling group
	idOrName := input.ScalingGroup
	if len(input.ScalingGroupId) != 0 {
		idOrName = input.ScalingGroupId
	}
	model, err := ScalingGroupManager.FetchByIdOrName(ctx, userCred, idOrName)
	if errors.Cause(err) == sql.ErrNoRows {
		return input, httperrors.NewInputParameterError("no such scaling group %s", idOrName)
	}
	if err != nil {
		return input, errors.Wrap(err, "ScalingGroupManager.FetchByIdOrName")
	}
	input.ScalingGroupId = model.GetId()

	if !utils.IsInStringArray(input.TriggerType, []string{api.TRIGGER_TIMING, api.TRIGGER_CYCLE, api.TRIGGER_ALARM}) {
		return input, httperrors.NewInputParameterError("unkown trigger type %s", input.TriggerType)
	}
	if !utils.IsInStringArray(input.Action, []string{api.ACTION_ADD, api.ACTION_REMOVE, api.ACTION_SET}) {
		return input, httperrors.NewInputParameterError("unkown scaling policy action %s", input.Action)
	}
	if !utils.IsInStringArray(input.Unit, []string{api.UNIT_ONE, api.UNIT_PERCENT}) {
		return input, httperrors.NewInputParameterError("unkown scaling policy unit %s", input.Unit)
	}
	trigger, err := ScalingPolicyManager.Trigger(&input)
	if err != nil {
		return input, errors.Wrap(err, "ScalingPolicyManager.Trigger")
	}
	input, err = trigger.ValidateCreateData(input)
	if err != nil {
		return input, httperrors.NewInputParameterError("%v", err)
	}
	return input, err
}

func (sp *SScalingPolicy) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// sp.Project must be same with sp.ScalingGroup
	sg, err := sp.ScalingGroup()
	if err != nil {
		return err
	}
	ownerId = sg.GetOwnerId()
	return sp.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (sp *SScalingPolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// do nothing
	sp.SetStatus(ctx, userCred, api.SP_STATUS_DELETING, "")
	return nil
}

func (sp *SScalingPolicy) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	trigger, err := sp.Trigger(nil)
	if err != nil {
		return errors.Wrap(err, "SScalingPolicy.Trigger")
	}
	if trigger != nil {
		err = trigger.UnRegister(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "IScalingTrigger.UnRegister")
		}
	}
	return sp.SResourceBase.Delete(ctx, userCred)
}

func (sp *SScalingPolicy) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	go func() {
		fail := func(sg *SScalingGroup, reason string) {
			if sg != nil {
				logclient.AddActionLogWithContext(ctx, sg, logclient.ACT_DELETE_SCALING_POLICY, reason, userCred, false)
			}
			sp.SetStatus(ctx, userCred, api.SP_STATUS_DELETE_FAILED, reason)
		}
		sg, err := sp.ScalingGroup()
		if err != nil {
			fail(nil, err.Error())
			return
		}
		err = sp.RealDelete(ctx, userCred)
		if err != nil {
			fail(sg, err.Error())
			return
		}
		logclient.AddActionLogWithContext(ctx, sg, logclient.ACT_DELETE_SCALING_POLICY, "", userCred, true)
	}()
}

func (spm *SScalingPolicyManager) Trigger(input *api.ScalingPolicyCreateInput) (IScalingTrigger, error) {
	tem := SScalingPolicy{TriggerType: input.TriggerType}
	return tem.Trigger(input)
}

func (sp *SScalingPolicy) Trigger(input *api.ScalingPolicyCreateInput) (IScalingTrigger, error) {
	log.Debugf("inset Trigger")
	if input != nil {
		switch sp.TriggerType {
		case api.TRIGGER_TIMING:
			return &SScalingTimer{
				SScalingPolicyBase: SScalingPolicyBase{sp.GetId()},
				STimer: STimer{
					Type:      api.TIMER_TYPE_ONCE,
					StartTime: input.Timer.ExecTime,
					EndTime:   input.Timer.ExecTime,
					NextTime:  input.Timer.ExecTime,
				},
			}, nil
		case api.TRIGGER_CYCLE:
			trigger := &SScalingTimer{
				SScalingPolicyBase: SScalingPolicyBase{sp.GetId()},
				STimer: STimer{
					Type:      input.CycleTimer.CycleType,
					Minute:    input.CycleTimer.Minute,
					Hour:      input.CycleTimer.Hour,
					StartTime: input.CycleTimer.StartTime,
					EndTime:   input.CycleTimer.EndTime,
					NextTime:  time.Time{},
				},
			}
			log.Debugf("setweekdays")
			trigger.SetWeekDays(input.CycleTimer.WeekDays)
			log.Debugf("setmonthdays")
			trigger.SetMonthDays(input.CycleTimer.MonthDays)
			trigger.Update(time.Now())
			return trigger, nil
		case api.TRIGGER_ALARM:
			return &SScalingAlarm{
				SScalingPolicyBase: SScalingPolicyBase{sp.GetId()},
				Cumulate:           input.Alarm.Cumulate,
				Cycle:              input.Alarm.Cycle,
				Indicator:          input.Alarm.Indicator,
				Wrapper:            input.Alarm.Wrapper,
				Operator:           input.Alarm.Operator,
				Value:              input.Alarm.Value,
				RealCumulate:       0,
				LastTriggerTime:    time.Now(),
			}, nil
		default:
			return nil, fmt.Errorf("unkown trigger type %s", sp.TriggerType)
		}
	}
	if len(sp.TriggerId) == 0 {
		return nil, nil
	}
	switch sp.TriggerType {
	case api.TRIGGER_TIMING, api.TRIGGER_CYCLE:
		model, err := ScalingTimerManager.FetchById(sp.TriggerId)
		if err != nil {
			return nil, errors.Wrap(err, "SScalingTimerManager.FetchById")
		}
		return model.(*SScalingTimer), nil
	case api.TRIGGER_ALARM:
		model, err := ScalingAlarmManager.FetchById(sp.TriggerId)
		if err != nil {
			return nil, errors.Wrap(err, "SScalingAlarmManager.FetchById")
		}
		return model.(*SScalingAlarm), nil
	default:
		return nil, fmt.Errorf("unkown trigger type %s", sp.TriggerType)
	}
}

func (sp *SScalingPolicy) PerformTrigger(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	if sp.Status != api.SP_STATUS_READY {
		return nil, httperrors.NewForbiddenError("Can't trigger scaling policy without status 'ready'")
	}

	var (
		triggerDesc IScalingTriggerDesc
		err         error
	)
	if sp.Enabled.IsFalse() {
		return nil, nil
	}
	sg, err := sp.ScalingGroup()
	if err != nil {
		return nil, errors.Wrap(err, "ScalingPolicy.ScalingGroup")
	}
	if sg.Enabled.IsFalse() {
		return nil, nil
	}

	manual, _ := data.Bool("manual")
	if manual {
		triggerDesc = SScalingManual{SScalingPolicyBase{sp.Id}}
	} else {
		trigger, err := sp.Trigger(nil)
		if err != nil {
			return nil, errors.Wrap(err, "fetch trigger failed")
		}
		if data.Contains("alarm_id") {
			alarmId, _ := data.GetString("alarm_id")
			if alarmId != trigger.(*SScalingAlarm).AlarmId {
				return nil, httperrors.NewInputParameterError("mismatched alarm id")
			}
		}
		if !trigger.IsTrigger() {
			return nil, nil
		}
		triggerDesc = trigger
	}
	err = sg.Scale(ctx, triggerDesc, sp, sp.CoolingTime)
	if err != nil {
		return nil, errors.Wrap(err, "ScalingPolicy.Scale")
	}
	sp.EventNotify(ctx, userCred)
	return nil, err
}

func (sp *SScalingPolicy) EventNotify(ctx context.Context, userCred mcclient.TokenCredential) {
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    sp,
		Action: notifyclient.ActionExecute,
	})
}

func (sp *SScalingPolicy) ScalingGroup() (*SScalingGroup, error) {
	model, err := ScalingGroupManager.FetchById(sp.ScalingGroupId)
	if err != nil {
		return nil, errors.Wrap(err, "ScalingGroupManager.FetchById")
	}
	return model.(*SScalingGroup), nil
}

type IScalingAction interface {
	Exec(int) int
	CheckCoolTime() bool
}

func (sp *SScalingPolicy) Exec(from int) int {
	diff := sp.Number
	if sp.Unit == api.UNIT_PERCENT {
		diff = diff * from / 100
	}
	switch sp.Action {
	case api.ACTION_ADD:
		return from + diff
	case api.ACTION_REMOVE:
		return from - diff
	case api.ACTION_SET:
		return diff
	default:
		return from
	}
}

func (sp *SScalingPolicy) CheckCoolTime() bool {
	if sp.TriggerType == api.TRIGGER_ALARM {
		return true
	}
	return false
}

func (sp *SScalingPolicy) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(sp, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (sp *SScalingPolicy) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(sp, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (sg *SScalingPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	sg.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	sg.SetStatus(ctx, userCred, api.SP_STATUS_CREATING, "")
	go func() {
		sp, err := sg.ScalingGroup()
		if err != nil {
			log.Errorf("Get ScalingGroup of ScalingPolicy '%s' failed: %s", sg.GetId(), err.Error())
		}
		createFailed := func(reason string) {
			logclient.AddActionLogWithContext(ctx, sp, logclient.ACT_CREATE_SCALING_POLICY, reason, userCred, false)
			sg.SetStatus(ctx, userCred, api.SP_STATUS_CREATE_FAILED, reason)
		}
		input := api.ScalingPolicyCreateInput{}
		err = data.Unmarshal(&input)
		if err != nil {
			createFailed(fmt.Sprintf("data.Unmarshal: %s", err.Error()))
			return
		}

		trigger, err := sg.Trigger(&input)
		if err != nil {
			createFailed(fmt.Sprintf("ScalingPolicy get trigger: %s", err.Error()))
			return
		}
		err = trigger.Register(ctx, userCred)
		if err != nil {
			createFailed(fmt.Sprintf("Trigger.Register: %s", err.Error()))
			return
		}
		logclient.AddActionLogWithContext(ctx, sp, logclient.ACT_CREATE_SCALING_POLICY, "", userCred, true)
		db.Update(sg, func() error {
			sg.TriggerId = trigger.TriggerId()
			sg.Status = api.SP_STATUS_READY
			sg.SetEnabled(true)
			return nil
		})
	}()
}
