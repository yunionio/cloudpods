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

package cloudprovider

import "strconv"

type SnapshotPolicyInput struct {
	RetentionDays  int
	RepeatWeekdays []int
	TimePoints     []int
	Name           string
	Desc           string
	Tags           map[string]string
}

func (spi *SnapshotPolicyInput) GetStringArrayRepeatWeekdays() []string {
	return toStringArray(spi.RepeatWeekdays)
}

func (spi *SnapshotPolicyInput) GetStringArrayTimePoints() []string {
	return toStringArray(spi.TimePoints)
}

func toStringArray(days []int) []string {
	ret := make([]string, len(days))
	for i := 0; i < len(days); i++ {
		ret[i] = strconv.Itoa(days[i])
	}
	return ret
}
