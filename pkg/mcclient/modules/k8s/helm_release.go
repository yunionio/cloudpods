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

package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Releases               *ReleaseManager
	MeterReleaseApps       *ReleaseAppManager
	ServicetreeReleaseApps *ReleaseAppManager
	NotifyReleaseApps      *ReleaseAppManager
	dummyReleaseApps       *ReleaseAppManager
)

type ReleaseManager struct {
	*NamespaceResourceManager
}

type ReleaseAppManager struct {
	*NamespaceResourceManager
}

func NewReleaseAppManager(keyword, keywordPlural string) *ReleaseAppManager {
	return &ReleaseAppManager{
		NamespaceResourceManager: NewNamespaceResourceManager(keyword, keywordPlural, NewColumns(), NewColumns()),
	}
}

func (m *ReleaseAppManager) Create(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.CreateInContext(session, params, dummyReleaseApps, "")
}

func init() {
	Releases = &ReleaseManager{
		NewNamespaceResourceManager("release", "releases", NewNamespaceCols("status", "type"), NewColumns())}
	dummyReleaseApps = NewReleaseAppManager("releaseapp", "releaseapps")
	MeterReleaseApps = NewReleaseAppManager("app_meter", "app_meters")
	ServicetreeReleaseApps = NewReleaseAppManager("app_servicetree", "app_servicetrees")
	NotifyReleaseApps = NewReleaseAppManager("app_notify", "app_notifies")
	modules.Register(Releases)
	modules.Register(MeterReleaseApps)
	modules.Register(ServicetreeReleaseApps)
	modules.Register(NotifyReleaseApps)
}
