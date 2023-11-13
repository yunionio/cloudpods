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

package yunionconf

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type BugReportManager struct {
	modulebase.ResourceManager
}

var (
	BugReport BugReportManager
)

func init() {
	BugReport = BugReportManager{modules.NewYunionConfManager("bug-report", "bug-report",
		[]string{},
		[]string{},
	)}
	modules.Register(&BugReport)
}

func (m BugReportManager) DoBugReportEnable(s *mcclient.ClientSession, _ jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return modulebase.Post(m.ResourceManager, s, "enable-bug-report", nil, "")
}

func (m BugReportManager) DoBugReportDisable(s *mcclient.ClientSession, _ jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return modulebase.Post(m.ResourceManager, s, "disable-bug-report", nil, "")
}

func (m BugReportManager) GetBugReportEnabled(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return modulebase.Get(m.ResourceManager, s, "bug-report-status", "")
}

func (m BugReportManager) SendBugReport(ctx context.Context, version, stack string, err error) (jsonutils.JSONObject, error) {
	msg := map[string]interface{}{
		"version": version,
		"stack":   stack,
		"message": err.Error(),
	}
	s := auth.GetAdminSession(ctx, "")
	return modulebase.Post(m.ResourceManager, s, "send-bug-report", jsonutils.Marshal(msg), "")
}
