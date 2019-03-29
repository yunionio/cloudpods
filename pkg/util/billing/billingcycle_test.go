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
	for _, cycleStr := range []string{"1H", "1D", "1W", "1M", "1Y"} {
		bc, err := ParseBillingCycle(cycleStr)
		if err != nil {
			t.Errorf("error parse %s: %s", cycleStr, err)
		} else {
			t.Logf("%s + %s = %s", now, bc.String(), bc.EndAt(now))
		}
	}
}
