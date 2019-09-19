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

package cronman

import (
	"context"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestSCronJobManager_AddRemoveJobs(t *testing.T) {
	manager := GetCronJobManager(false)
	testFunc := func(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {}
	manager.AddJobAtIntervals("Test1", time.Second*100, testFunc)

	manager.AddJobAtIntervals("Test2", time.Second*100, testFunc)
	manager.AddJobEveryFewDays("Test3", 1, 1, 1, 1, testFunc, false)
	manager.AddJobEveryFewDays("Test4", 1, 1, 1, 1, testFunc, false)
	manager.AddJobEveryFewDays("Test5", 1, 1, 1, 1, testFunc, false)
	manager.Start()
	t.Logf("Jobs \n%s", manager.String())
	manager.Remove("Test1")
	manager.Remove("Test2")
	manager.Remove("Test3")
	manager.AddJobAtIntervals("Test6", time.Second*100, testFunc)
	manager.AddJobEveryFewDays("Test7", 1, 1, 1, 1, testFunc, false)
	t.Logf("Jobs \n%s", manager.String())
}
