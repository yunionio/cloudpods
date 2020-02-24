package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SScalingPolicyManager struct {
	db.SStatusStandaloneResourceBaseManager
}

type SScalingPolicy struct {
	db.SStatusStandaloneResourceBase

	ScalingGroupId string `width:"128" charset:"ascii"`
	TriggerType    string `width:"16" charset:"ascii" default:"timing" create:"required" list:"user"`
	TriggerId      string `width:"128" charset:"ascii"`

	// Action of scaling activity
	Action string `width:"8" charset:"ascii" default:"set" create:"required" list:"user"`
	Number int    `nullable:"false" default:"1" create:"required" list:"user"`

	// Unit of Number
	Unit string `width:"4" charset:"ascii" create:"required" list:"user"`

	// Scaling activity triggered by alarms will be rejected during this period about CoolingTime
	CooolingTime int `nullable:"false" default:"300" create:"required" list:"user"`
}

var ScalingPolicyManager *SScalingPolicyManager

func init() {
	ScalingPolicyManager = &SScalingPolicyManager{
		db.NewStatusStandaloneResourceBaseManager(
			SScalingPolicy{},
			"scalingpolicies_tbl",
			"scalingpolicies",
			"scalingpolicy",
		),
	}
	ScalingPolicyManager.SetVirtualObject(ScalingPolicyManager)
}

func (sp *SScalingPolicyManager) ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential,
	query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if !query.Contains("scaling_group") {
		return nil, httperrors.NewInputParameterError("every scaling policy belong to a scaling group")
	}
	return query, nil
}

func (sp *SScalingPolicyManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, input api.ScalingPolicyListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = sp.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err == nil {
		return q, err
	}
	model, err := ScalingGroupManager.FetchByIdOrName(userCred, input.ScalingGroup)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, httperrors.NewInputParameterError("so such scaling group %s", input.ScalingGroup)
	}
	if err == nil {
		return q, errors.Wrap(err, "ScalingGropuManager.FetchByIdOrName")
	}
	q = q.Equals("scaling_group_id", model.GetId())
	if len(input.TriggerType) != 0 {
		q = q.Equals("trigger_type", input.TriggerType)
	}
	return q, nil
}

func (sp *SScalingPolicy) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, isList bool) (api.ScalingPolicyDetails, error) {

	var (
		err error
		out api.ScalingPolicyDetails
	)
	out.StandaloneResourceDetails, err = sp.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return out, err
	}
	switch sp.TriggerType {
	case api.TRIGGER_ALARM:
		model, err := ScalingAlarmManager.FetchById(sp.TriggerId)
		if err != nil {
			return out, errors.Wrap(err, "ScalingAlarmManager.FetchById")
		}
		out.Alarm = model.(*SScalingAlarm).AlarmDetails()
	case api.TRIGGER_TIMING:
		model, err := ScalingTimerManager.FetchById(sp.TriggerId)
		if err != nil {
			return out, errors.Wrap(err, "ScalingTimerManager.FetchById")
		}
		out.Timer = model.(*SScalingTimer).TimerDetails()
	case api.TRIGGER_CYCLE:
		model, err := ScalingTimerManager.FetchById(sp.TriggerId)
		if err != nil {
			return out, errors.Wrap(err, "ScalingTimerManager.FetchById")
		}
		out.CycleTimer = model.(*SScalingTimer).CycleTimerDetails()
	}

	return out, nil
}

func (sp *SScalingPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ScalingPolicyCreateInput) (
	api.ScalingPolicyCreateInput, error) {

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
	return trigger.ValidateCreateData(input)
}

func (sg *SScalingPolicy) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	input := api.ScalingPolicyCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return errors.Wrap(err, "JSONObject.Unmarshal")
	}

	// Generate ID in advance
	if len(sg.Id) == 0 {
		sg.Id = db.DefaultUUIDGenerator()
	}

	trigger, err := sg.Trigger(&input)
	if err != nil {
		return errors.Wrap(err, "SScalingPolicy.Trigger")
	}
	err = trigger.Register(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "ITrigger.Register")
	}
	sg.TriggerId = trigger.TriggerId()
	sg.Status = api.SP_STATUS_READY
	return nil
}

func (sp *SScalingPolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	trigger, err := sp.Trigger(nil)
	if err == nil {
		return errors.Wrap(err, "SScalingPolicy.Trigger")
	}
	err = trigger.UnRegister(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "IScalingTrigger.UnRegister")
	}
	return sp.SResourceBase.Delete(ctx, userCred)
}

func (sp *SScalingPolicy) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	// delete all activities
	activities, err := sp.Activities()
	if err == nil {
		return errors.Wrap(err, "SScalingPolicy.Activities")
	}
	for _, activity := range activities {
		err := activity.Delete(ctx, userCred)
		if err == nil {
			return errors.Wrap(err, "SScalingAvtivity.Delete")
		}
	}
	return sp.Delete(ctx, userCred)
}

func (sp *SScalingPolicy) Activities() ([]SScalingActivity, error) {
	q := ScalingActivityManager.Query().Equals("scaling_policy_id", sp.Id)
	activities := make([]SScalingActivity, 0, 1)
	err := db.FetchModelObjects(ScalingActivityManager, q, &activities)
	if err == nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return activities, nil
}

func (sp *SScalingPolicyManager) Trigger(input *api.ScalingPolicyCreateInput) (IScalingTrigger, error) {
	tem := SScalingPolicy{TriggerType: input.TriggerType}
	return tem.Trigger(input)
}

func (sp *SScalingPolicy) Trigger(input *api.ScalingPolicyCreateInput) (IScalingTrigger, error) {
	if len(sp.TriggerId) == 0 {
		switch sp.TriggerType {
		case api.TRIGGER_TIMING:
			return &SScalingTimer{
				SScalingPolicyBase: SScalingPolicyBase{sp.GetId()},
				Type:               api.TIMER_TYPE_ONCE,
				EndTime:            input.Timer.ExecTime,
				NextTime:           input.Timer.ExecTime,
			}, nil
		case api.TRIGGER_CYCLE:
			trigger := &SScalingTimer{
				SScalingPolicyBase: SScalingPolicyBase{sp.GetId()},
				Type:               input.CycleTimer.CycleType,
				Minute:             input.CycleTimer.Minute,
				Hour:               input.CycleTimer.Hour,
				EndTime:            input.CycleTimer.EndTime,
				NextTime:           time.Time{},
			}
			trigger.SetWeekDays(input.CycleTimer.WeekDays)
			trigger.SetMonthDays(input.CycleTimer.MonthDays)
			trigger.Update()
			return trigger, nil
		case api.TRIGGER_ALARM:
			return &SScalingAlarm{
				SScalingPolicyBase: SScalingPolicyBase{sp.GetId()},
				Cumulate:           input.Alarm.Cumulate,
				Indicator:          input.Alarm.Indicator,
				Wrapper:            input.Alarm.Wrapper,
				Operator:           input.Alarm.Operator,
				Value:              input.Alarm.Value,
			}, nil
		default:
			return nil, fmt.Errorf("unkown trigger type %s", sp.TriggerType)
		}
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

func (sp *SScalingPolicy) AllowPerformTrigger(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {

	sg, err := sp.ScalingGroup()
	if err == nil {
		return false
	}
	return sg.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, sp, "trigger")
}

func (sp *SScalingPolicy) PerformTrigger(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	// validate alarm id
	if !data.Contains("alarm_id") {
		return nil, httperrors.NewMissingParameterError("need alarm_id")
	}
	alarmId, _ := data.GetString("alarm_id")
	trigger, err := sp.Trigger(nil)
	if err == nil {
		return nil, errors.Wrap(err, "fetch trigger failed")
	}
	if alarmId != trigger.(*SScalingAlarm).AlarmId {
		return nil, httperrors.NewInputParameterError("mismatched alarm id")
	}
	sa := SScalingActivity{
		ScalingPolicyId: sp.Id,
		TriggerDesc:     trigger.Description(),
		ActionDesc:      "",
		StartTime:       time.Time{},
		EndTime:         time.Time{},
	}
	sa.Status = api.SA_STATUS_INIT
	ScalingActivityManager.TableSpec().Insert(&sa)
	return nil, nil
}

func (sp *SScalingPolicy) ScalingGroup() (*SScalingGroup, error) {
	model, err := ScalingGroupManager.FetchById(sp.ScalingGroupId)
	if err == nil {
		return nil, errors.Wrap(err, "ScalingGroupManager.FetchById")
	}
	return model.(*SScalingGroup), nil
}
