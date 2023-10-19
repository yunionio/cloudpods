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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/util/bitmap"
)

type STimer struct {
	// Cycle type
	Type string `width:"8" charset:"ascii"`
	// 0-59
	Minute int `nullable:"false"`
	// 0-23
	Hour int `nullable:"false"`
	// 0-7 1 is Monday 0 is unlimited
	WeekDays uint8 `nullable:"false"`
	// 0-31 0 is unlimited
	MonthDays uint32 `nullable:"false"`

	// StartTime represent the start time of this timer
	StartTime time.Time
	// EndTime represent deadline of this timer
	EndTime time.Time
	// NextTime represent the time timer should bell
	NextTime  time.Time `index:"true"`
	IsExpired bool
}

// Update will update the SScalingTimer
func (st *STimer) Update(now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	if !now.Before(st.EndTime) {
		st.IsExpired = true
		return
	}
	if now.Before(st.StartTime) {
		now = st.StartTime
	}
	if !st.NextTime.Before(now) {
		return
	}

	newNextTime := time.Date(now.Year(), now.Month(), now.Day(), st.Hour, st.Minute, 0, 0, time.UTC).In(now.Location())
	if now.After(newNextTime) {
		newNextTime = newNextTime.AddDate(0, 0, 1)
	}
	switch {
	case st.WeekDays != 0:
		// week
		nowDay, weekdays := int(newNextTime.Weekday()), st.GetWeekDays()
		if nowDay == 0 {
			nowDay = 7
		}

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
			newNextTime = time.Date(suitTime.Year(), suitTime.Month(), monthdays[index], suitTime.Hour(),
				suitTime.Minute(), 0, 0, suitTime.Location())
			break
		}
	default:
		// day

	}
	log.Debugf("The final NextTime: %s", newNextTime)
	st.NextTime = newNextTime
	if st.NextTime.After(st.EndTime) {
		st.IsExpired = true
	}
}

// MonthDaySum calculate the number of month's days
func (st *STimer) MonthDaySum(t time.Time) int {
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

func (st *STimer) GetWeekDays() []int {
	return bitmap.Uint2IntArray(uint32(st.WeekDays))
}

func (st *STimer) GetMonthDays() []int {
	return bitmap.Uint2IntArray(st.MonthDays)
}

func (st *STimer) SetWeekDays(days []int) {
	st.WeekDays = uint8(bitmap.IntArray2Uint(days))
}

func (st *STimer) SetMonthDays(days []int) {
	st.MonthDays = bitmap.IntArray2Uint(days)
}

func (st *STimer) TimerDetails() api.TimerDetails {
	return api.TimerDetails{ExecTime: st.EndTime}
}

func (st *STimer) CycleTimerDetails() api.CycleTimerDetails {
	out := api.CycleTimerDetails{
		Minute:    st.Minute,
		Hour:      st.Hour,
		WeekDays:  st.GetWeekDays(),
		MonthDays: st.GetMonthDays(),
		StartTime: st.StartTime,
		EndTime:   st.EndTime,
		CycleType: st.Type,
	}
	return out
}

func checkTimerCreateInput(in api.TimerCreateInput) (api.TimerCreateInput, error) {
	now := time.Now()
	if now.After(in.ExecTime) {
		return in, fmt.Errorf("exec_time is earlier than now")
	}
	return in, nil
}

var (
	timerDescTable = i18n.Table{}
	TIMERLANG      = "timerLang"
)

func init() {
	timerDescTable.Set("timerLang", i18n.NewTableEntry().EN("en").CN("cn"))
}

func (st *STimer) Description(ctx context.Context) string {
	lang := timerDescTable.Lookup(ctx, TIMERLANG)
	switch lang {
	case "en":
		return st.descEnglish()
	case "cn":
		return st.descChinese()
	}
	return ""
}

var (
	wdsCN = []string{"", "一", "二", "三", "四", "五", "六", "日"}
	wdsEN = []string{"", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	zone  = time.Now().Local().Location()
	//zone  = time.FixedZone("GMT", 8*3600)
)

func (st *STimer) descChinese() string {
	format := "2006-01-02 15:04:05"
	var prefix string
	switch st.Type {
	case api.TIMER_TYPE_ONCE:
		return fmt.Sprintf("单次 %s触发", st.StartTime.In(zone).Format(format))
	case api.TIMER_TYPE_DAY:
		prefix = "每天"
	case api.TIMER_TYPE_WEEK:
		wds := st.GetWeekDays()
		weekDays := make([]string, len(wds))
		for i := range wds {
			weekDays[i] = fmt.Sprintf("星期%s", wdsCN[wds[i]])
		}
		prefix = fmt.Sprintf("每周 【%s】", strings.Join(weekDays, "｜"))
	case api.TIMER_TYPE_MONTH:
		mns := st.GetMonthDays()
		monthDays := make([]string, len(mns))
		for i := range mns {
			monthDays[i] = fmt.Sprintf("%d号", mns[i])
		}
		prefix = fmt.Sprintf("每月 【%s】", strings.Join(monthDays, "｜"))
	}
	return fmt.Sprintf("%s %02d:%02d触发 有效时间为%s至%s", prefix, st.Hour, st.Minute, st.StartTime.In(zone).Format(format), st.EndTime.In(zone).Format(format))
}

func (st *STimer) descEnglish() string {
	var detail string
	format := "2006-01-02 15:04:05"
	switch st.Type {
	case api.TIMER_TYPE_ONCE:
		return st.EndTime.In(zone).Format(format)
	case api.TIMER_TYPE_DAY:
		detail = fmt.Sprintf("%d:%d every day", st.Hour, st.Minute)
	case api.TIMER_TYPE_WEEK:
		detail = st.weekDaysDesc()
	case api.TIMER_TYPE_MONTH:
		detail = st.monthDaysDesc()
	}
	if st.EndTime.IsZero() {
		return detail
	}
	return fmt.Sprintf("%s, from %s to %s", detail, st.StartTime.In(zone).Format(format), st.EndTime.In(zone).Format(format))
}

func (st *STimer) weekDaysDesc() string {
	if st.WeekDays == 0 {
		return ""
	}
	var desc strings.Builder
	wds := st.GetWeekDays()
	i := 0
	desc.WriteString(fmt.Sprintf("%d:%d every %s", st.Hour, st.Minute, wdsEN[wds[i]]))
	for i++; i < len(wds)-1; i++ {
		desc.WriteString(", ")
		desc.WriteString(wdsEN[wds[i]])
	}
	if i == len(wds)-1 {
		desc.WriteString(" and ")
		desc.WriteString(wdsEN[wds[i]])
	}
	return desc.String()
}

func (st *STimer) monthDaysDesc() string {
	if st.MonthDays == 0 {
		return ""
	}
	var desc strings.Builder
	mds := st.GetMonthDays()
	i := 0
	desc.WriteString(fmt.Sprintf("%d:%d on the %d%s", st.Hour, st.Minute, mds[i], st.dateSuffix(mds[i])))
	for i++; i < len(mds)-1; i++ {
		desc.WriteString(", ")
		desc.WriteString(strconv.Itoa(mds[i]))
		desc.WriteString(st.dateSuffix(mds[i]))
	}
	if i == len(mds)-1 {
		desc.WriteString(" and ")
		desc.WriteString(strconv.Itoa(mds[i]))
		desc.WriteString(st.dateSuffix(mds[i]))
	}
	desc.WriteString(" of each month")
	return desc.String()
}

func (st *STimer) dateSuffix(date int) string {
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

func checkCycleTimerCreateInput(in api.CycleTimerCreateInput) (api.CycleTimerCreateInput, error) {
	now := time.Now()
	if in.Minute < 0 || in.Minute > 59 {
		return in, fmt.Errorf("minute should between 0 and 59")
	}
	if in.Hour < 0 || in.Hour > 23 {
		return in, fmt.Errorf("hour should between 0 and 23")
	}
	switch in.CycleType {
	case api.TIMER_TYPE_HOUR:
		if in.CycleHour <= 0 || in.CycleHour >= 24 {
			return in, fmt.Errorf("cycle_hour should between 0 and 23")
		}
		in.WeekDays = []int{}
		in.MonthDays = []int{}
	case api.TIMER_TYPE_DAY:
		in.WeekDays = []int{}
		in.MonthDays = []int{}
	case api.TIMER_TYPE_WEEK:
		if len(in.WeekDays) == 0 {
			return in, fmt.Errorf("week_days should not be empty")
		}
		in.MonthDays = []int{}
	case api.TIMER_TYPE_MONTH:
		if len(in.MonthDays) == 0 {
			return in, fmt.Errorf("month_days should not be empty")
		}
		in.WeekDays = []int{}
	default:
		return in, fmt.Errorf("unkown cycle type %s", in.CycleType)
	}
	if now.After(in.EndTime) {
		return in, fmt.Errorf("end_time is earlier than now")
	}
	return in, nil
}
