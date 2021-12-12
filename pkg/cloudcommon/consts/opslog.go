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

package consts

import "time"

var (
	globalOpsLogEnabled       = true
	splitableMaxDurationHours = 24 * 30 // 30 days, 1month
	splitableMaxKeepMonths    = 6       // 6 months, half year
)

func DisableOpsLog() {
	globalOpsLogEnabled = false
}

func OpsLogEnabled() bool {
	return globalOpsLogEnabled
}

func SetSplitableMaxKeepMonths(cnt int) {
	splitableMaxKeepMonths = cnt
}

func SetSplitableMaxDurationHours(h int) {
	splitableMaxDurationHours = h
}

func SplitableMaxKeepMonths() int {
	return splitableMaxKeepMonths
}

func SplitableMaxDuration() time.Duration {
	return time.Hour * time.Duration(splitableMaxDurationHours)
}
