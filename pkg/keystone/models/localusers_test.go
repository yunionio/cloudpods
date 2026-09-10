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
)

func TestAuthThrottleRemain(t *testing.T) {
	now := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	usr := &SLocalUser{}

	if got := usr.AuthThrottleRemain(5, now); got != 0 {
		t.Fatalf("empty record: got %v", got)
	}

	usr.FailedAuthCount = 5
	usr.FailedAuthAt = now.Add(-time.Second)
	if got := usr.AuthThrottleRemain(5, now); got != 0 {
		t.Fatalf("at threshold: got %v", got)
	}

	usr.FailedAuthCount = 6
	if got := usr.AuthThrottleRemain(5, now); got != 29*time.Second {
		t.Fatalf("first over threshold: got %v", got)
	}

	usr.FailedAuthCount = 7
	if got := usr.AuthThrottleRemain(5, now); got != 59*time.Second {
		t.Fatalf("second over threshold: got %v", got)
	}

	usr.FailedAuthAt = now.Add(-2 * time.Minute)
	if got := usr.AuthThrottleRemain(5, now); got != 0 {
		t.Fatalf("cooldown elapsed: got %v", got)
	}

	usr.FailedAuthCount = 20
	usr.FailedAuthAt = now.Add(-time.Second)
	if got := usr.AuthThrottleRemain(5, now); got != authThrottleCap-time.Second {
		t.Fatalf("capped cooldown: got %v", got)
	}

	if got := usr.AuthThrottleRemain(0, now); got != 0 {
		t.Fatalf("lock disabled: got %v", got)
	}
}
