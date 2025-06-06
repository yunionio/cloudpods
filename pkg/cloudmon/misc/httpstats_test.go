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

package misc

import (
	"testing"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/timeutils"
)

func TestUnmarshalHttpStats(t *testing.T) {
	input := `{"duration.2XX":4532188.710192006, "duration.4XX":196870.22898800002, "duration.5XX":32787.14909299999, "hit.2XX":88678, "hit.4XX":1748, "hit.5XX":179, "paths":[{"duration.2XX":150, "duration.4XX":250, "duration.5XX":310, "hit.2XX":1500, "hit.4XX":2500, "hit.5XX":3500, "method":"GET", "path":"/servers", "name":"list_servers"}, {"duration.2XX":150, "duration.4XX":250, "duration.5XX":350, "hit.2XX":1500, "hit.4XX":2500, "hit.5XX":3500, "method":"POST", "path":"/servers", "name":"create_servers"}]}`
	inputJson, err := jsonutils.ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("json: %s", inputJson.PrettyString())
	var stats sApiHttpStats
	err = inputJson.Unmarshal(&stats)
	if err != nil {
		t.Fatal(err)
	}
	if stats.HttpCode2xx != 4532188.710192006 {
		t.Fatalf("duration.2XX: %f", stats.HttpCode2xx)
	}
	if stats.HttpCode4xx != 196870.22898800002 {
		t.Fatalf("duration.4XX: %f", stats.HttpCode4xx)
	}
	if stats.HttpCode5xx != 32787.14909299999 {
		t.Fatalf("duration.5XX: %f", stats.HttpCode5xx)
	}
	if stats.HitHttpCode2xx != 88678 {
		t.Fatalf("hit.2XX: %d", stats.HitHttpCode2xx)
	}
	if stats.HitHttpCode4xx != 1748 {
		t.Fatalf("hit.4XX: %d", stats.HitHttpCode4xx)
	}
	if stats.HitHttpCode5xx != 179 {
		t.Fatalf("hit.5XX: %d", stats.HitHttpCode5xx)
	}
}

func TestHttpStats(t *testing.T) {
	cases := []struct {
		prevTime  time.Time
		nowTime   time.Time
		prevStats *sApiHttpStats
		nowStats  sApiHttpStats
	}{
		{
			nowTime: func() time.Time {
				tm, _ := timeutils.ParseTimeStr("2025-06-01 00:01:00")
				return tm
			}(),
			nowStats: sApiHttpStats{
				SHttpStats: SHttpStats{
					HttpCode2xx:    400,
					HttpCode4xx:    700,
					HttpCode5xx:    1000,
					HitHttpCode2xx: 4000,
					HitHttpCode4xx: 7000,
					HitHttpCode5xx: 10000,
				},
				Paths: []SHttpStats{
					{
						HttpCode2xx:    150,
						HttpCode4xx:    250,
						HttpCode5xx:    310,
						HitHttpCode2xx: 1500,
						HitHttpCode4xx: 2500,
						HitHttpCode5xx: 3500,
						Method:         "GET",
						Path:           "/servers",
						Name:           "list_servers",
					},
					{
						HttpCode2xx:    150,
						HttpCode4xx:    250,
						HttpCode5xx:    350,
						HitHttpCode2xx: 1500,
						HitHttpCode4xx: 2500,
						HitHttpCode5xx: 3500,
						Method:         "POST",
						Path:           "/servers",
						Name:           "create_servers",
					},
					{
						HttpCode2xx:    100,
						HttpCode4xx:    200,
						HttpCode5xx:    300,
						HitHttpCode2xx: 1000,
						HitHttpCode4xx: 2000,
						HitHttpCode5xx: 3000,
						Method:         "PUT",
						Path:           "/servers/*",
						Name:           "update_servers",
					},
				},
			},
		},
		{
			prevTime: func() time.Time {
				tm, _ := timeutils.ParseTimeStr("2025-06-01 00:00:00")
				return tm
			}(),
			nowTime: func() time.Time {
				tm, _ := timeutils.ParseTimeStr("2025-06-01 00:01:00")
				return tm
			}(),
			prevStats: &sApiHttpStats{
				SHttpStats: SHttpStats{
					HttpCode2xx:    200,
					HttpCode4xx:    400,
					HttpCode5xx:    600,
					HitHttpCode2xx: 2000,
					HitHttpCode4xx: 4000,
					HitHttpCode5xx: 6000,
				},
				Paths: []SHttpStats{
					{
						HttpCode2xx:    100,
						HttpCode4xx:    200,
						HttpCode5xx:    300,
						HitHttpCode2xx: 1000,
						HitHttpCode4xx: 2000,
						HitHttpCode5xx: 3000,
						Method:         "GET",
						Path:           "/servers",
						Name:           "list_servers",
					},
					{
						HttpCode2xx:    100,
						HttpCode4xx:    200,
						HttpCode5xx:    300,
						HitHttpCode2xx: 1000,
						HitHttpCode4xx: 2000,
						HitHttpCode5xx: 3000,
						Method:         "POST",
						Path:           "/servers",
						Name:           "create_servers",
					},
				},
			},
			nowStats: sApiHttpStats{
				SHttpStats: SHttpStats{
					HttpCode2xx:    400,
					HttpCode4xx:    700,
					HttpCode5xx:    1000,
					HitHttpCode2xx: 4000,
					HitHttpCode4xx: 7000,
					HitHttpCode5xx: 10000,
				},
				Paths: []SHttpStats{
					{
						HttpCode2xx:    150,
						HttpCode4xx:    250,
						HttpCode5xx:    310,
						HitHttpCode2xx: 1500,
						HitHttpCode4xx: 2500,
						HitHttpCode5xx: 3500,
						Method:         "GET",
						Path:           "/servers",
						Name:           "list_servers",
					},
					{
						HttpCode2xx:    150,
						HttpCode4xx:    250,
						HttpCode5xx:    350,
						HitHttpCode2xx: 1500,
						HitHttpCode4xx: 2500,
						HitHttpCode5xx: 3500,
						Method:         "POST",
						Path:           "/servers",
						Name:           "create_servers",
					},
					{
						HttpCode2xx:    100,
						HttpCode4xx:    200,
						HttpCode5xx:    300,
						HitHttpCode2xx: 1000,
						HitHttpCode4xx: 2000,
						HitHttpCode5xx: 3000,
						Method:         "PUT",
						Path:           "/servers/*",
						Name:           "update_servers",
					},
				},
			},
		},
	}
	for _, c := range cases {
		var prev *sHttpStatsSnapshot
		if c.prevStats != nil {
			prev = c.prevStats.convertSnapshot(c.prevTime)
		}
		curr := c.nowStats.convertSnapshot(c.nowTime)
		diff := calculateHttpStatsDiff(prev, curr)
		metrics := diff.metrics("test", "test", "test", "test")
		t.Log(jsonutils.Marshal(metrics).PrettyString())
	}
}
