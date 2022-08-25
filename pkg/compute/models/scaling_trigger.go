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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	monapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
)

type IScalingTriggerDesc interface {
	TriggerDescription() string
}

type IScalingTrigger interface {
	IScalingTriggerDesc

	// ValidateCreateData check and verify the input when creating SScalingPolicy
	ValidateCreateData(input api.ScalingPolicyCreateInput) (api.ScalingPolicyCreateInput, error)

	// Register
	Register(ctx context.Context, userCred mcclient.TokenCredential) error
	UnRegister(ctx context.Context, userCred mcclient.TokenCredential) error
	TriggerId() string
	IsTrigger() bool
}

type SScalingManual struct {
	SScalingPolicyBase
}

func (sm SScalingManual) TriggerDescription() string {
	name := sm.ScalingPolicyId
	sp, _ := sm.ScalingPolicy()
	if sp != nil {
		name = sp.Name
	}
	return fmt.Sprintf(`A user request to execute scaling policy "%s"`, name)
}

type SScalingPolicyBase struct {
	ScalingPolicyId string `width:"36" charset:"ascii"`
}

func (spb *SScalingPolicyBase) ScalingGroup() (*SScalingGroup, error) {
	q := ScalingPolicyManager.Query().In("id", ScalingPolicyManager.Query("scaling_group_id").Equals("id",
		spb.ScalingPolicyId).SubQuery())
	var sg SScalingGroup
	err := q.First(&sg)
	return &sg, err
}

func (spb *SScalingPolicyBase) ScalingPolicy() (*SScalingPolicy, error) {
	model, err := ScalingPolicyManager.FetchById(spb.ScalingPolicyId)
	if err != nil {
		return nil, errors.Wrap(err, "ScalingPolicyManager.FetchById")
	}
	return model.(*SScalingPolicy), nil
}

type SScalingTimerManager struct {
	db.SStandaloneResourceBaseManager
}

type SScalingTimer struct {
	db.SStandaloneResourceBase

	SScalingPolicyBase

	STimer
}

type SScalingAlarmManager struct {
	db.SStandaloneResourceBaseManager
}

// 1st, 2nd, 3rd

type SScalingAlarm struct {
	db.SStandaloneResourceBase

	SScalingPolicyBase

	// ID of alarm config in alarm service
	AlarmId string `width:"128" charset:"ascii"`

	// Trigger when the cumulative count is reached
	Cumulate  int
	Cycle     int
	Indicator string `width:"32" charset:"ascii"`

	// Wrapper instruct how to calculate collective data based on individual data
	Wrapper  string `width:"16" charset:"ascii"`
	Operator string `width:"2" charset:"ascii"`

	Value float64

	// Real-time cumulate number
	RealCumulate int `default:"0"`
	// Last trigger time
	LastTriggerTime time.Time
}

var (
	ScalingTimerManager *SScalingTimerManager
	ScalingAlarmManager *SScalingAlarmManager
)

func init() {
	ScalingTimerManager = &SScalingTimerManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SScalingTimer{},
			"scalingtimers_tbl",
			"scalingtimer",
			"scalingtimers",
		),
	}
	ScalingTimerManager.SetVirtualObject(ScalingTimerManager)

	ScalingAlarmManager = &SScalingAlarmManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SScalingAlarm{},
			"scalingalarms_tbl",
			"scalingalarm",
			"scalingalarms",
		),
	}
	ScalingAlarmManager.SetVirtualObject(ScalingAlarmManager)
}

func (sa *SScalingAlarm) AlarmDetails() api.ScalingAlarmDetails {
	return api.ScalingAlarmDetails{
		Cumulate:  sa.Cumulate,
		Cycle:     sa.Cycle,
		Indicator: sa.Indicator,
		Wrapper:   sa.Wrapper,
		Operator:  sa.Operator,
		Value:     sa.Value,
	}
}

func (st *SScalingTimer) ValidateCreateData(input api.ScalingPolicyCreateInput) (api.ScalingPolicyCreateInput, error) {
	var err error
	if input.TriggerType == api.TRIGGER_TIMING {
		input.Timer, err = checkTimerCreateInput(input.Timer)
	} else {
		input.CycleTimer, err = checkCycleTimerCreateInput(input.CycleTimer)
	}
	if err != nil {
		return input, httperrors.NewInputParameterError("%v", err)
	}
	return input, nil
}

func (st *SScalingTimer) Register(ctx context.Context, userCred mcclient.TokenCredential) error {
	// insert
	st.Update(time.Time{})
	err := ScalingTimerManager.TableSpec().Insert(ctx, st)
	if err != nil {
		return errors.Wrap(err, "STableSpec.Insert")
	}
	return nil
}

func (st *SScalingTimer) UnRegister(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := st.Delete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "SScalingTimer.Delete")
	}
	return nil
}

func (st *SScalingTimer) TriggerId() string {
	return st.GetId()
}

var cstSh, _ = time.LoadLocation("Asia/Shanghai")

func (st *SScalingTimer) TriggerDescription() string {
	detail := st.descEnglish()
	name := st.ScalingPolicyId
	sp, _ := st.ScalingPolicy()
	if sp != nil {
		name = sp.Name
	}
	return fmt.Sprintf(`Schedule task(%s) execute scaling policy "%s"`, detail, name)
}

func (st *SScalingTimer) IsTrigger() bool {
	return true
}

func (sa *SScalingAlarm) ValidateCreateData(input api.ScalingPolicyCreateInput) (api.ScalingPolicyCreateInput, error) {
	if len(input.Alarm.Operator) == 0 {
		input.Alarm.Operator = api.OPERATOR_GT
	}
	if input.Alarm.Cycle == 0 {
		input.Alarm.Cycle = 300
	}
	if !utils.IsInStringArray(input.Alarm.Operator, []string{api.OPERATOR_GT, api.OPERATOR_LT}) {
		return input, httperrors.NewInputParameterError("unkown operator in alarm %s", input.Alarm.Operator)
	}
	if !utils.IsInStringArray(input.Alarm.Indicator, []string{api.INDICATOR_CPU, api.INDICATOR_DISK_READ,
		api.INDICATOR_DISK_WRITE, api.INDICATOR_FLOW_INTO, api.INDICATOR_FLOW_OUT}) {
		return input, httperrors.NewInputParameterError("unkown indicator in alarm %s", input.Alarm.Indicator)
	}
	if !utils.IsInStringArray(input.Alarm.Wrapper, []string{api.WRAPPER_MIN, api.WRAPPER_MAX, api.WRAPPER_AVER}) {
		return input, httperrors.NewInputParameterError("unkown wrapper in alarm %s", input.Alarm.Wrapper)
	}
	if input.Alarm.Cycle < 300 {
		return input, httperrors.NewInputParameterError("the min value of cycle in alarm is 300")
	}
	return input, nil
}

func (spm *SScalingPolicyManager) NotificationID(session *mcclient.ClientSession) (string, error) {
	var notificationID = ""
	params := jsonutils.NewDict()
	params.Set("type", jsonutils.NewString(monapi.AlertNotificationTypeAutoScaling))

	result, err := monitor.Notifications.List(session, params)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return "", errors.Wrap(err, "Notifications.List")
	}
	if result.Total != 0 {
		notificationID, _ = result.Data[0].GetString("id")
		return notificationID, nil
	}
	// To create new one
	conTrue, conFalse := true, false
	ncinput := monapi.NotificationCreateInput{
		Name:                  fmt.Sprintf("as-%s-%s", session.GetDomainName(), session.GetProjectName()),
		Type:                  monapi.AlertNotificationTypeAutoScaling,
		IsDefault:             false,
		SendReminder:          &conFalse,
		DisableResolveMessage: &conTrue,
		Settings:              jsonutils.NewDict(),
	}
	ret, err := monitor.Notifications.Create(session, jsonutils.Marshal(ncinput))
	if err != nil {
		return "", errors.Wrap(err, "Notification.Create")
	}
	notificationID, _ = ret.GetString("id")
	return notificationID, nil
}

func (sa *SScalingAlarm) Register(ctx context.Context, userCred mcclient.TokenCredential) error {
	sp, err := sa.ScalingPolicy()
	if err != nil {
		return err
	}
	session := auth.GetSession(ctx, userCred, "")
	notificationID, err := ScalingPolicyManager.NotificationID(session)
	if err != nil {
		return errors.Wrap(err, "ScalingPolicyManager.NotificationID")
	}
	// create Alert
	config, err := sa.generateAlertConfig(sp)
	if err != nil {
		return errors.Wrap(err, "ScalingAlarm.generateAlertConfig")
	}
	alert, err := monitor.Alerts.DoCreate(session, config)
	if err != nil {
		return errors.Wrap(err, "create Alert failed")
	}
	alarmId, _ := alert.GetString("id")
	// detach
	params := jsonutils.NewDict()
	params.Set("scaling_policy_id", jsonutils.NewString(sa.ScalingPolicyId))
	detachParams := jsonutils.NewDict()
	detachParams.Set("params", params)
	_, err = monitor.Alertnotification.Attach(session, alarmId, notificationID, detachParams)
	if err != nil {
		monitor.Alerts.Delete(session, alarmId, jsonutils.NewDict())
		return errors.Wrap(err, "attach alert with notification")
	}
	sa.AlarmId = alarmId

	// insert
	err = ScalingAlarmManager.TableSpec().Insert(ctx, sa)
	if err != nil {
		return errors.Wrap(err, "STableSpec.Insert")
	}

	return nil
}

type sTableField struct {
	Table string
	Field string
}

var indicatorMap = map[string]sTableField{
	api.INDICATOR_CPU:        {"vm_cpu", "usage_active"},
	api.INDICATOR_DISK_WRITE: {"vm_diskio", "write_bps"},
	api.INDICATOR_DISK_READ:  {"vm_diskio", "read_bps"},
	api.INDICATOR_FLOW_INTO:  {"vm_netio", "bps_recv"},
	api.INDICATOR_FLOW_OUT:   {"vm_netio", "bps_sent"},
}

var alertConfigUsedBy = "scaling_group"

func (sa *SScalingAlarm) generateAlertConfig(sp *SScalingPolicy) (*monitor.AlertConfig, error) {
	config, err := monitor.NewAlertConfig(fmt.Sprintf("sp-%s", sp.Id), fmt.Sprintf("%ds", sa.Cycle), true)
	if err != nil {
		return nil, err
	}
	config.UsedBy = alertConfigUsedBy
	cond := config.Condition("telegraf", indicatorMap[sa.Indicator].Table).Avg()
	log.Debugf("alarm: %#v", sa)

	switch sa.Operator {
	case api.OPERATOR_LT:
		cond = cond.LT(sa.Value)
	case api.OPERATOR_GT:
		cond = cond.GT(sa.Value)
	}
	q := cond.Query().From("1h")
	sel := q.Selects().Select(indicatorMap[sa.Indicator].Field)
	switch sa.Wrapper {
	case api.WRAPPER_AVER:
		sel = sel.MEAN()
	case api.WRAPPER_MAX:
		sel = sel.MAX()
	case api.WRAPPER_MIN:
		sel = sel.MIN()
	}
	q.Where().Equal("vm_scaling_group_id", sp.ScalingGroupId)
	q.GroupBy().TAG("*").FILL_NULL()
	return config, nil
}

func (sa *SScalingAlarm) UnRegister(ctx context.Context, userCred mcclient.TokenCredential) error {
	session := auth.GetSession(ctx, userCred, "")
	_, err := monitor.Alerts.Delete(session, sa.AlarmId, jsonutils.NewDict())
	if err != nil {
		return errors.Wrap(err, "Alerts.Delete")
	}
	err = sa.Delete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "SSCalingAlarm.Delete")
	}
	return nil
}

func (sa *SScalingAlarm) TriggerId() string {
	return sa.GetId()
}

func (sa *SScalingAlarm) TriggerDescription() string {
	name := sa.ScalingPolicyId
	sp, _ := sa.ScalingPolicy()
	if sp != nil {
		name = sp.Name
	}
	return fmt.Sprintf(
		`Alarm task(the %s %s of the instance is %s than %f%s) execute scaling policy "%s"`,
		descs[sa.Wrapper], descs[sa.Indicator], descs[sa.Operator],
		sa.Value, units[sa.Indicator], name,
	)
}

func (sa *SScalingAlarm) IsTrigger() (is bool) {
	realCumulate := sa.RealCumulate
	lastTriggerTime := sa.LastTriggerTime
	now := time.Now()
	if lastTriggerTime.Add(time.Duration(sa.Cycle) * 2 * time.Second).Before(now) {
		realCumulate = 1
	} else {
		realCumulate += 1
	}
	lastTriggerTime = now
	if realCumulate == sa.Cumulate {
		is = true
		realCumulate = 0
	}
	_, err := db.Update(sa, func() error {
		sa.RealCumulate = realCumulate
		sa.LastTriggerTime = lastTriggerTime
		return nil
	})
	if err != nil {
		log.Errorf("db.Update in ScalingAlarm.IsTrigger failed: %s", err.Error())
	}
	return
}

var descs = map[string]string{
	api.INDICATOR_CPU:        "CPU utilization",
	api.INDICATOR_MEM:        "memory utilization",
	api.INDICATOR_DISK_READ:  "disk read rate",
	api.INDICATOR_DISK_WRITE: "disk write rate",
	api.INDICATOR_FLOW_INTO:  "network inflow rate",
	api.INDICATOR_FLOW_OUT:   "network outflow rate",
	api.WRAPPER_MAX:          "maximum",
	api.WRAPPER_MIN:          "minimum",
	api.WRAPPER_AVER:         "average",
	api.OPERATOR_GT:          "greater",
	api.OPERATOR_LT:          "less",
}

var units = map[string]string{
	api.INDICATOR_CPU:        "%",
	api.INDICATOR_MEM:        "%",
	api.INDICATOR_DISK_READ:  "kB/s",
	api.INDICATOR_DISK_WRITE: "kB/s",
	api.INDICATOR_FLOW_INTO:  "KB/s",
	api.INDICATOR_FLOW_OUT:   "KB/s",
}
