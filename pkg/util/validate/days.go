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

package validate

import (
	"fmt"
	"sort"
)

// DaysValidate sort days and check if days is out of range [min, max] or has repeated member
func DaysCheck(days []int, min, max int) ([]int, error) {
	if len(days) == 0 {
		return days, nil
	}
	sort.Ints(days)

	if days[0] < min || days[len(days)-1] > max {
		return days, fmt.Errorf("Out of range")
	}

	for i := 1; i < len(days); i++ {
		if days[i] == days[i-1] {
			return days, fmt.Errorf("Has repeat day %v", days)
		}
	}
	return days, nil
}
