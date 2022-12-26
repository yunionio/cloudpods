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

package billing

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
)

type TBillingCycleUnit string

const (
	BillingCycleMinute = TBillingCycleUnit("I")
	BillingCycleHour   = TBillingCycleUnit("H")
	BillingCycleDay    = TBillingCycleUnit("D")
	BillingCycleWeek   = TBillingCycleUnit("W")
	BillingCycleMonth  = TBillingCycleUnit("M")
	BillingCycleYear   = TBillingCycleUnit("Y")
)

var (
	ErrInvalidBillingCycle = errors.New("invalid billing cycle")
	ErrInvalidDuration     = errors.New("invalid duration")
)

type SBillingCycle struct {
	AutoRenew bool
	Count     int
	Unit      TBillingCycleUnit
}

func ParseBillingCycle(cycleStr string) (SBillingCycle, error) {
	cycle := SBillingCycle{}
	if len(cycleStr) < 2 {
		return cycle, ErrInvalidBillingCycle
	}
	switch cycleStr[len(cycleStr)-1:] {
	case string(BillingCycleMinute), strings.ToLower(string(BillingCycleMinute)):
		cycle.Unit = BillingCycleMinute
	case string(BillingCycleHour), strings.ToLower(string(BillingCycleHour)):
		cycle.Unit = BillingCycleHour
	case string(BillingCycleDay), strings.ToLower(string(BillingCycleDay)):
		cycle.Unit = BillingCycleDay
	case string(BillingCycleWeek), strings.ToLower(string(BillingCycleWeek)):
		cycle.Unit = BillingCycleWeek
	case string(BillingCycleMonth), strings.ToLower(string(BillingCycleMonth)):
		cycle.Unit = BillingCycleMonth
	case string(BillingCycleYear), strings.ToLower(string(BillingCycleYear)):
		cycle.Unit = BillingCycleYear
	default:
		return cycle, ErrInvalidBillingCycle
	}
	val, err := strconv.Atoi(cycleStr[:len(cycleStr)-1])
	if err != nil {
		log.Errorf("invalid BillingCycle string %s: %v", cycleStr, err)
		return cycle, ErrInvalidBillingCycle
	}
	cycle.Count = val
	return cycle, nil
}

// parse duration to minute unit billing cycle
func DurationToBillingCycle(dur time.Duration) SBillingCycle {
	return SBillingCycle{
		Unit:  BillingCycleMinute,
		Count: int(dur.Minutes()),
	}
}

func (cycle *SBillingCycle) String() string {
	return fmt.Sprintf("%d%s", cycle.Count, cycle.Unit)
}

func (cycle SBillingCycle) EndAt(tm time.Time) time.Time {
	if tm.IsZero() {
		tm = time.Now().UTC()
	}
	switch cycle.Unit {
	case BillingCycleMinute:
		return tm.Add(time.Minute * time.Duration(cycle.Count))
	case BillingCycleHour:
		return tm.Add(time.Hour * time.Duration(cycle.Count))
	case BillingCycleDay:
		return tm.AddDate(0, 0, cycle.Count)
	case BillingCycleWeek:
		return tm.AddDate(0, 0, cycle.Count*7)
	case BillingCycleMonth:
		return tm.AddDate(0, cycle.Count, 0)
	case BillingCycleYear:
		return tm.AddDate(cycle.Count, 0, 0)
	default:
		return tm.Add(time.Hour * time.Duration(cycle.Count))
	}
}

func minuteStart(tm time.Time) time.Time {
	return time.Date(
		tm.Year(),
		tm.Month(),
		tm.Day(),
		tm.Hour(),
		tm.Minute(),
		0,
		0,
		tm.Location(),
	)
}

func hourStart(tm time.Time) time.Time {
	return time.Date(
		tm.Year(),
		tm.Month(),
		tm.Day(),
		tm.Hour(),
		0,
		0,
		0,
		tm.Location(),
	)
}

func dayStart(tm time.Time) time.Time {
	return time.Date(
		tm.Year(),
		tm.Month(),
		tm.Day(),
		0,
		0,
		0,
		0,
		tm.Location(),
	)
}

func weekStart(tm time.Time) time.Time {
	tm = dayStart(tm)
	dayOff := int(tm.Weekday()) - 1
	if dayOff < 0 {
		dayOff += 7
	}
	return tm.Add(-time.Duration(dayOff) * time.Hour * 24)
}

func monthStart(tm time.Time) time.Time {
	return time.Date(
		tm.Year(),
		tm.Month(),
		1,
		0,
		0,
		0,
		0,
		tm.Location(),
	)
}

func yearStart(tm time.Time) time.Time {
	return time.Date(
		tm.Year(),
		time.January,
		1,
		0,
		0,
		0,
		0,
		tm.Location(),
	)
}

func (cycle SBillingCycle) LatestLastStart(tm time.Time) time.Time {
	if tm.IsZero() {
		tm = time.Now().UTC()
	}
	switch cycle.Unit {
	case BillingCycleMinute:
		return minuteStart(tm)
	case BillingCycleHour:
		return hourStart(tm)
	case BillingCycleDay:
		return dayStart(tm)
	case BillingCycleWeek:
		return weekStart(tm)
	case BillingCycleMonth:
		return monthStart(tm)
	case BillingCycleYear:
		return yearStart(tm)
	default: // hour
		return hourStart(tm)
	}
}

func (cycle SBillingCycle) TimeString(tm time.Time) string {
	if tm.IsZero() {
		tm = time.Now().UTC()
	}
	switch cycle.Unit {
	case BillingCycleMinute:
		return tm.Format("200601021504")
	case BillingCycleHour:
		return tm.Format("2006010215")
	case BillingCycleDay:
		return tm.Format("20060102")
	case BillingCycleWeek:
		return tm.Format("20060102w")
	case BillingCycleMonth:
		return tm.Format("200601")
	case BillingCycleYear:
		return tm.Format("2006")
	default: // hour
		return tm.Format("2006010215")
	}
}

func (cycle SBillingCycle) Duration() time.Duration {
	now := time.Now().UTC()
	endAt := cycle.EndAt(now)
	return endAt.Sub(now)
}

func (cycle *SBillingCycle) GetDays() int {
	switch cycle.Unit {
	case BillingCycleMinute:
		return cycle.Count / 24 / 60
	case BillingCycleHour:
		return cycle.Count / 24
	case BillingCycleDay:
		return cycle.Count
	case BillingCycleWeek:
		return cycle.Count * 7
	default:
		return 0
	}
}

func (cycle *SBillingCycle) GetWeeks() int {
	switch cycle.Unit {
	case BillingCycleMinute:
		return cycle.Count / 7 / 24 / 60
	case BillingCycleHour:
		return cycle.Count / 7 / 24
	case BillingCycleDay:
		return cycle.Count / 7
	case BillingCycleWeek:
		return cycle.Count
	default:
		return 0
	}
}

func (cycle *SBillingCycle) GetMonths() int {
	switch cycle.Unit {
	case BillingCycleMonth:
		return cycle.Count
	case BillingCycleYear:
		return cycle.Count * 12
	default:
		return 0
	}
}

func (cycle *SBillingCycle) GetYears() int {
	switch cycle.Unit {
	case BillingCycleMonth:
		if cycle.Count%12 == 0 {
			return cycle.Count / 12
		}
		return 0
	case BillingCycleYear:
		return cycle.Count
	default:
		return 0
	}
}

func (cycle *SBillingCycle) IsValid() bool {
	return cycle.Unit != "" && cycle.Count > 0
}
