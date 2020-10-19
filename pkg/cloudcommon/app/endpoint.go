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

package app

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/syncman/watcher"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SEndpointChangeManager struct {
	watcher.SInformerSyncManager
}

func newEndpointChangeManager() *SEndpointChangeManager {
	man := &SEndpointChangeManager{}
	man.InitSync(man)
	man.FirstSync()
	return man
}

func (man *SEndpointChangeManager) DoSync(first bool) (time.Duration, error) {
	// reauth to refresh endpoint list
	auth.ReAuth()
	return time.Hour * 2, nil
}

func (man *SEndpointChangeManager) NeedSync(dat *jsonutils.JSONDict) bool {
	return true
}

func (man *SEndpointChangeManager) Name() string {
	return "EndpointChangeManager"
}
