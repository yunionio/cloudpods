package models

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/bitmap"
	"yunion.io/x/pkg/errors"
)

type IScalingTrigger interface {
	// ValidateCreateData check and verify the input when creating SScalingPolicy
	ValidateCreateData(input api.ScalingPolicyCreateInput) (api.ScalingPolicyCreateInput, error)

	// Register
	Register(ctx context.Context, userCred mcclient.TokenCredential) error
	UnRegister(ctx context.Context, userCred mcclient.TokenCredential) error
	TriggerId() string
	Description() string
}

type SScalingPolicyBase struct {
	ScalingPolicyId string `width:"128" charset:"ascii"`
}

func (spb *SScalingPolicyBase) ScalingGroup() (*SScalingGroup, error) {
	q := ScalingPolicyManager.Query().In("id", ScalingPolicyManager.Query("scaling_group_id").Equals("id",
		spb.ScalingPolicyId).SubQuery())
	var sg SScalingGroup
	err := q.First(&sg)
	return &sg, err
}

func (spb *SScalingPolicyBase) ScalingPlicy() (*SScalingPolicy, error) {
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

	// Timer type
	Type string `width:"8" charset:"ascii"`

	// 0-59
	Minute int `nullable:"false"`

	// 0-23
	Hour int `nullable:"false"`

	// 0-7 1 is Monday 0 is unlimited
	WeekDays uint8 `nullable:"false"`

	// 0-31 0 is unlimited
	MonthDays uint32 `nullable:"false"`

	// EndTime represent deadline of this timer
	EndTime time.Time

	// NextTime represent the time timer should bell
	NextTime  time.Time `index:"true"`
	IsExpired bool
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
	Indicator string `width:"32" charset:"ascii"`

	// Wrapper instruct how to calculate collective data based on individual data
	Wrapper  string `width:"16" charset:"ascii"`
	Operator string `width:"2" charset:"ascii"`

	// Value is a percentage describing the threshold
	Value int
}

var (
	ScalingTimerManager *SScalingTimerManager
	ScalingAlarmManager *SScalingAlarmManager
)

func init() {
	ScalingTimerManager = &SScalingTimerManager{
		db.NewStandaloneResourceBaseManager(
			SScalingTimer{},
			"scalingtimers_tbl",
			"scalingtimers",
			"scalingtimer",
		),
	}
}

func (st *SScalingTimer) GetWeekDays() []int {
	return bitmap.Uint2IntArray(uint32(st.WeekDays))
}

func (st *SScalingTimer) GetMonthDays() []int {
	return bitmap.Uint2IntArray(st.MonthDays)
}

func (st *SScalingTimer) SetWeekDays(days []int) {
	st.WeekDays = uint8(bitmap.IntArray2Uint(days))
}

func (st *SScalingTimer) SetMonthDays(days []int) {
	st.MonthDays = bitmap.IntArray2Uint(days)
}

// Update will update the SScalingTimer
func (st *SScalingTimer) Update() {
	now := time.Now()
	if !now.Before(st.EndTime) {
		st.IsExpired = true
		return
	}
	if now.Before(st.NextTime) {
		return
	}

	newNextTime := time.Date(now.Year(), now.Month(), now.Day(), st.Hour, st.Minute, 0, 0, now.Location())
	if now.After(newNextTime) {
		newNextTime.AddDate(0, 0, 1)
	}
	switch {
	case st.WeekDays != 0:
		// week
		nowDay, weekdays := int(newNextTime.Weekday()), st.GetWeekDays()

		// weekdays[0]+7 is for the case that all time nodes has been missed in this week
		weekdays = append(weekdays, weekdays[0]+7)
		index := sort.SearchInts(weekdays, nowDay)
		newNextTime = newNextTime.AddDate(0, 0, weekdays[index]-nowDay)
	case st.MonthDays != 0:
		// month
		monthdays := st.GetMonthDays()
		suitTime := newNextTime
		for {
			day := suitTime.Day()
			index := sort.SearchInts(monthdays, day)
			if index == len(monthdays) || monthdays[index] > st.MonthDaySum(suitTime) {
				// set suitTime as the first day of next month
				suitTime = suitTime.AddDate(0, 1, -suitTime.Day()+1)
				continue
			}
			newNextTime = time.Date(suitTime.Year(), suitTime.Month(), suitTime.Day(), suitTime.Hour(),
				suitTime.Minute(), 0, 0, suitTime.Location())
		}
	default:
		// day

	}
	st.NextTime = newNextTime
}

// MonthDaySum calculate the number of month's days
func (st *SScalingTimer) MonthDaySum(t time.Time) int {
	year, month := t.Year(), t.Month()
	monthDays := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month != 2 {
		return monthDays[2]
	}
	if year%4 != 0 || (year%100 == 0 && year%400 != 0) {
		return 28
	}
	return 29
}

func (st *SScalingTimer) TimerDetails() api.ScalingTimerDetails {
	return api.ScalingTimerDetails{ExecTime: st.EndTime}
}

func (st *SScalingTimer) CycleTimerDetails() api.ScalingCycleTimerDetails {
	out := api.ScalingCycleTimerDetails{
		Minute:    st.Minute,
		Hour:      st.Hour,
		WeekDays:  st.GetWeekDays(),
		MonthDays: st.GetMonthDays(),
		EndTime:   st.EndTime,
		CycleType: st.Type,
	}
	return out
}

func (sa *SScalingAlarm) AlarmDetails() api.ScalingAlarmDetails {
	return api.ScalingAlarmDetails{
		Cumulate:  sa.Cumulate,
		Indicator: sa.Indicator,
		Wrapper:   sa.Wrapper,
		Operator:  sa.Operator,
		Value:     sa.Value,
	}
}

func (st *SScalingTimer) ValidateCreateData(input api.ScalingPolicyCreateInput) (api.ScalingPolicyCreateInput, error) {
	now := time.Now()
	if input.TriggerType == api.TRIGGER_TIMING {
		if now.After(input.Timer.ExecTime) {
			return input, fmt.Errorf("exec_time is earlier than now")
		}
		return input, nil
	}
	if input.CycleTimer.Minute < 0 || input.CycleTimer.Minute > 59 {
		return input, fmt.Errorf("minute should between 0 and 59")
	}
	if input.CycleTimer.Hour < 0 || input.CycleTimer.Hour > 23 {
		return input, fmt.Errorf("hour should between 0 and 23")
	}
	switch input.CycleTimer.CycleType {
	case api.TIMER_TYPE_DAY:
		input.CycleTimer.WeekDays = []int{}
		input.CycleTimer.MonthDays = []int{}
	case api.TIMER_TYPE_WEEK:
		if len(input.CycleTimer.WeekDays) == 0 {
			return input, fmt.Errorf("week_days should not be empty")
		}
		input.CycleTimer.MonthDays = []int{}
	case api.TIMER_TYPE_MONTH:
		if len(input.CycleTimer.MonthDays) == 0 {
			return input, fmt.Errorf("month_days should not be empty")
		}
		input.CycleTimer.WeekDays = []int{}
	default:
		return input, fmt.Errorf("unkown cycle type %s", input.CycleTimer.CycleType)
	}
	if now.After(input.CycleTimer.EndTime) {
		return input, fmt.Errorf("end_time is earlier than now")
	}
	return input, nil
}

func (st *SScalingTimer) Register(ctx context.Context, userCred mcclient.TokenCredential) error {
	// insert
	st.Update()
	err := ScalingTimerManager.TableSpec().Insert(st)
	if err != nil {
		return errors.Wrap(err, "STableSpec.Insert")
	}
	return nil
}

func (st *SScalingTimer) UnRegister(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := st.Delete(ctx, userCred)
	if err == nil {
		return errors.Wrap(err, "SScalingTimer.Delete")
	}
	return nil
}

func (st *SScalingTimer) TriggerId() string {
	return st.GetId()
}

func (st *SScalingTimer) Description() string {
	var detail string
	switch st.Type {
	case api.TIMER_TYPE_ONCE:
		detail = st.EndTime.String()
	case api.TIMER_TYPE_DAY:
		detail = fmt.Sprintf("%d:%d every day", st.Hour, st.Minute)
	case api.TIMER_TYPE_WEEK:
		detail = st.WeekDaysDesc()
	case api.TIMER_TYPE_MONTH:
		detail = st.MonthDaysDesc()
	}
	return fmt.Sprintf("Schedule task(%s)", detail)
}

func (sa *SScalingAlarm) ValidateCreateData(input api.ScalingPolicyCreateInput) (api.ScalingPolicyCreateInput, error) {
	return input, nil
}

func (sa *SScalingAlarm) Register(ctx context.Context, userCred mcclient.TokenCredential) error {
	// insert
	err := ScalingAlarmManager.TableSpec().Insert(sa)
	if err != nil {
		return errors.Wrap(err, "STableSpec.Insert")
	}
	// todo: register alarm rule to alarm service
	return nil
}

func (sa *SScalingAlarm) UnRegister(ctx context.Context, userCred mcclient.TokenCredential) error {
	// todo: unregister alarm rule to alarm service

	err := sa.Delete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "SSCalingAlarm.Delete")
	}
	return nil
}

func (sa *SScalingAlarm) TriggerId() string {
	return sa.GetId()
}

func (sa *SScalingAlarm) Description() string {
	return fmt.Sprintf(
		"Alarm task(the %s %s of the instance is %s than %d%s)",
		descs[sa.Wrapper], descs[sa.Indicator], descs[sa.Operator],
		sa.Value, units[sa.Indicator],
	)
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
	api.OPERATOR_GE:          "greater",
	api.OPERATOR_LE:          "less",
}

var units = map[string]string{
	api.INDICATOR_CPU:        "%",
	api.INDICATOR_MEM:        "%",
	api.INDICATOR_DISK_READ:  "kb/s",
	api.INDICATOR_DISK_WRITE: "kb/s",
	api.INDICATOR_FLOW_INTO:  "KB/s",
	api.INDICATOR_FLOW_OUT:   "KB/s",
}

var weekDays = []string{"", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

func (st *SScalingTimer) WeekDaysDesc() string {
	if st.WeekDays == 0 {
		return ""
	}
	var desc strings.Builder
	wds := st.GetWeekDays()
	i := 0
	desc.WriteString(fmt.Sprintf("%d:%d every %s", st.Hour, st.Minute, weekDays[wds[i]]))
	for i++; i < len(wds)-1; i++ {
		desc.WriteString(", ")
		desc.WriteString(weekDays[wds[i]])
	}
	if i == len(wds)-1 {
		desc.WriteString(" and ")
		desc.WriteString(weekDays[wds[i]])
	}
	return desc.String()
}

func (st *SScalingTimer) MonthDaysDesc() string {
	if st.MonthDays == 0 {
		return ""
	}
	var desc strings.Builder
	mds := st.GetMonthDays()
	i := 0
	desc.WriteString(fmt.Sprintf("%d:%d on the %d%s", st.Hour, st.Minute, mds[i], dateSuffix(mds[i])))
	for i++; i < len(mds)-1; i++ {
		desc.WriteString(", ")
		desc.WriteString(strconv.Itoa(mds[i]))
		desc.WriteString(dateSuffix(mds[i]))
	}
	if i == len(mds)-1 {
		desc.WriteString(" and ")
		desc.WriteString(strconv.Itoa(mds[i]))
		desc.WriteString(dateSuffix(mds[i]))
	}
	desc.WriteString(" of each month")
	return desc.String()
}

func dateSuffix(date int) string {
	var ret string
	switch date {
	case 1:
		ret = "st"
	case 2:
		ret = "nd"
	case 3:
		ret = "rd"
	default:
		ret = "th"
	}
	return ret
}
