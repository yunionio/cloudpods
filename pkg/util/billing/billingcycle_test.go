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
	"testing"
	"time"
)

func TestParseBillingCycle(t *testing.T) {
	now := time.Now().UTC()
	for _, cycleStr := range []string{"1H", "2H", "1D", "1W", "2W", "3W", "4W", "1M", "2M", "1Y", "2Y"} {
		bc, err := ParseBillingCycle(cycleStr)
		if err != nil {
			t.Errorf("error parse %s: %s", cycleStr, err)
		} else {
			t.Logf("%s: %s + %s = %s Weeks: %d Months: %d", bc.String(), now, bc.String(), bc.EndAt(now), bc.GetWeeks(), bc.GetMonths())
		}
	}
}

func TestLatestLastStart(t *testing.T) {
	cases := []struct {
		tm   string
		bc   string
		want string
	}{
		{
			tm:   "2020-05-16T00:00:00Z",
			bc:   "1m",
			want: "2020-05-01T00:00:00Z",
		},
		{
			tm:   "2020-05-16T23:34:02Z",
			bc:   "1h",
			want: "2020-05-16T23:00:00Z",
		},
		{
			tm:   "2020-05-16T23:34:02Z",
			bc:   "1i",
			want: "2020-05-16T23:34:00Z",
		},
		{
			tm:   "2020-05-16T23:34:02Z",
			bc:   "1d",
			want: "2020-05-16T00:00:00Z",
		},
		{
			tm:   "2020-05-16T23:34:02Z",
			bc:   "1w",
			want: "2020-05-11T00:00:00Z",
		},
		{
			tm:   "2020-05-16T23:34:02Z",
			bc:   "1y",
			want: "2020-01-01T00:00:00Z",
		},
	}
	for _, c := range cases {
		tm, _ := time.Parse(time.RFC3339, c.tm)
		bc, _ := ParseBillingCycle(c.bc)
		got := bc.LatestLastStart(tm)
		want, _ := time.Parse(time.RFC3339, c.want)
		if got != want {
			t.Errorf("bc: %s want: %s got: %s", c.bc, want, got)
		}
	}
}

func TestTimeString(t *testing.T) {
	cases := []struct {
		tm   string
		bc   string
		want string
	}{
		{
			tm:   "2020-05-16T23:23:54Z",
			bc:   "1m",
			want: "202005",
		},
	}
	for _, c := range cases {
		tm, _ := time.Parse(time.RFC3339, c.tm)
		bc, _ := ParseBillingCycle(c.bc)
		got := bc.TimeString(tm)
		if c.want != got {
			t.Errorf("bc: %s TimeString want: %s got: %s", c.bc, c.want, got)
		}
	}
}
