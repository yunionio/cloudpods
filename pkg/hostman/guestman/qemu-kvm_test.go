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

package guestman

import (
	"testing"
)

func TestSKVMGuestInstance_getPid(t *testing.T) {
	manager := NewGuestManager(nil, "/opt/cloud/workspace/servers")
	s := NewKVMGuestInstance("05b787e9-b78e-4ebc-8128-04f55d37306f", manager)
	t.Logf("Guest is ->> %d", s.GetPid())
}
