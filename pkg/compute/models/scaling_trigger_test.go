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
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/bitmap"
)

func TestSTimer_Update(t *testing.T) {
	loc := time.UTC
	ins := []struct {
		Timer       *STimer
		UpdateTimes int
	}{
		{
			Timer: &STimer{
				Hour:      7,
				Minute:    21,
				Type:      compute.TIMER_TYPE_DAY,
				StartTime: time.Date(2020, 4, 9, 7, 18, 8, 0, loc),
				NextTime:  time.Date(2020, 4, 9, 7, 21, 0, 0, loc),
				EndTime:   time.Date(2020, 4, 11, 7, 18, 8, 0, loc),
			},
			UpdateTimes: 2,
		},
		{
			Timer: &STimer{
				Hour:      7,
				Minute:    21,
				WeekDays:  uint8(bitmap.IntArray2Uint([]int{1, 5, 7})),
				Type:      compute.TIMER_TYPE_WEEK,
				StartTime: time.Date(2020, 4, 9, 7, 18, 8, 0, loc),
				NextTime:  time.Date(2020, 4, 9, 7, 21, 0, 0, loc),
				EndTime:   time.Date(2020, 4, 14, 7, 18, 8, 0, loc),
			},
			UpdateTimes: 4,
		},
		{
			Timer: &STimer{
				Hour:      7,
				Minute:    21,
				MonthDays: bitmap.IntArray2Uint([]int{1, 10, 30}),
				Type:      compute.TIMER_TYPE_MONTH,
				StartTime: time.Date(2020, 4, 9, 7, 18, 8, 0, loc),
				NextTime:  time.Date(2020, 4, 9, 7, 21, 0, 0, loc),
				EndTime:   time.Date(2020, 5, 2, 7, 18, 8, 0, loc),
			},
			UpdateTimes: 4,
		},
	}
	wants := [][]time.Time{
		{
			time.Date(2020, 4, 10, 7, 21, 0, 0, loc),
			{},
		},
		{
			time.Date(2020, 4, 10, 7, 21, 0, 0, loc),
			time.Date(2020, 4, 12, 7, 21, 0, 0, loc),
			time.Date(2020, 4, 13, 7, 21, 0, 0, loc),
			{},
		},
		{
			time.Date(2020, 4, 10, 7, 21, 0, 0, loc),
			time.Date(2020, 4, 30, 7, 21, 0, 0, loc),
			time.Date(2020, 5, 1, 7, 21, 0, 0, loc),
			{},
		},
	}
	wrapper := func(timer *STimer) time.Time {
		fakeNow := timer.NextTime
		if !fakeNow.IsZero() {
			fakeNow = fakeNow.Add(time.Minute)
		}
		timer.Update(fakeNow)
		if timer.IsExpired {
			return time.Time{}
		}
		return timer.NextTime
	}
	for i := range ins {
		for updateTimes := 0; updateTimes < ins[i].UpdateTimes; updateTimes++ {
			out := wrapper(ins[i].Timer)
			if out.IsZero() && wants[i][updateTimes].IsZero() {
				continue
			}
			if out.Equal(wants[i][updateTimes]) {
				continue
			}
			t.Fatalf("index: %d, updateTimes: %d, want: %s, real: %s", i, updateTimes, wants[i][updateTimes], out)
		}
	}
}
