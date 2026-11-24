// Copyright 2026 Yunion
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

package llm

import (
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

// DashboardPanelManager talks to llm custom handlers under /dashboard_panels/*.
type DashboardPanelManager struct {
	modulebase.ResourceManager
}

var (
	DashboardPanel DashboardPanelManager
)

func init() {
	DashboardPanel = DashboardPanelManager{
		ResourceManager: modules.NewLLMManager("dashboard_panel", "dashboard_panels",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&DashboardPanel)
}

func (m *DashboardPanelManager) post(s *mcclient.ClientSession, action string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := "/" + m.URLPath() + "/" + strings.Trim(action, "/")
	// Align with ResourceManager.PerformClassAction: KeywordPlural envelope.
	wrapped := jsonutils.NewDict()
	if body != nil {
		wrapped.Set(m.KeywordPlural, body)
	}
	return modulebase.Post(m.ResourceManager, s, path, wrapped, m.KeywordPlural)
}

// Generate calls POST /dashboard_panels/generate.
func (m *DashboardPanelManager) Generate(s *mcclient.ClientSession, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.post(s, "generate", body)
}

// Validate calls POST /dashboard_panels/validate.
func (m *DashboardPanelManager) Validate(s *mcclient.ClientSession, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.post(s, "validate", body)
}

// Save calls POST /dashboard_panels/save.
func (m *DashboardPanelManager) Save(s *mcclient.ClientSession, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.post(s, "save", body)
}
