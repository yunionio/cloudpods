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

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
)

func TestSSnapshotPolicy_Key(t *testing.T) {
	cases := []struct {
		in   *SSnapshotPolicy
		want uint64
	}{
		{
			&SSnapshotPolicy{
				RepeatWeekdays: 0,
				TimePoints:     0,
				RetentionDays:  7,
				IsActivated:    tristate.True,
			},
			15 + 2,
		},
		{
			&SSnapshotPolicy{
				RepeatWeekdays: 11,
			},
			11<<56 + 2,
		},
		{
			&SSnapshotPolicy{
				TimePoints: 234,
			},
			234<<24 + 2,
		},
		{
			&SSnapshotPolicy{
				RetentionDays: -1,
			},
			0,
		},
		{
			&SSnapshotPolicy{
				RepeatWeekdays: 13,
				TimePoints:     123,
				RetentionDays:  7,
			},
			936748724556660752,
		},
	}

	for i, c := range cases {
		g := c.in.Key()
		if c.want != g {
			log.Errorf("the %d case, want %d, get %d", i, c.want, g)
		}
	}
}
