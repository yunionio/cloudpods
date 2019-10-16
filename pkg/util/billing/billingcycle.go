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
)

type SBillingCycle struct {
	Count int
	Unit  TBillingCycleUnit
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
