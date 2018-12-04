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
	BillingCycleHour  = TBillingCycleUnit("H")
	BillingCycleDay   = TBillingCycleUnit("D")
	BillingCycleWeek  = TBillingCycleUnit("W")
	BillingCycleMonth = TBillingCycleUnit("M")
	BillingCycleYear  = TBillingCycleUnit("Y")
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
		log.Errorf("invalid BillingCycle string %s: %", cycleStr, err)
		return cycle, ErrInvalidBillingCycle
	}
	cycle.Count = val
	return cycle, nil
}

func (cycle *SBillingCycle) String() string {
	return fmt.Sprintf("%d%s", cycle.Count, cycle.Unit)
}

func (cycle *SBillingCycle) EndAt(tm time.Time) time.Time {
	if tm.IsZero() {
		tm = time.Now().UTC()
	}
	switch cycle.Unit {
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

func (cycle *SBillingCycle) GetDays() int {
	switch cycle.Unit {
	case BillingCycleHour:
		if cycle.Count%24 == 0 {
			return cycle.Count / 24
		}
		return 0
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
	case BillingCycleHour:
		if cycle.Count%(7*24) == 0 {
			return cycle.Count / (7 * 24)
		}
		return 0
	case BillingCycleDay:
		if cycle.Count%7 == 0 {
			return cycle.Count / 7
		}
		return 0
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
