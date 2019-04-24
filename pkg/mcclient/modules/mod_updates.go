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

package modules

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SUpdateManager struct {
	ResourceManager
}

var (
	Updates SUpdateManager
)

func (this *SUpdateManager) DoUpdate(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	this.PerformAction(s, "", "", params)
	return ret, nil
}

func init() {

	Updates = SUpdateManager{NewAutoUpdateManager("update", "updates",
		// user view
		[]string{"localVersion", "remoteVersion", "status", "updateAvailable"},
		[]string{}, // admin view
	)}

	register(&Updates)
}
