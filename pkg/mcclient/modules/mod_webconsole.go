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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	WebConsole WebConsoleManager
)

func init() {
	WebConsole = WebConsoleManager{NewWebConsoleManager()}

	register(&WebConsole)
}

type WebConsoleManager struct {
	modulebase.ResourceManager
}

func NewWebConsoleManager() modulebase.ResourceManager {
	return modulebase.ResourceManager{BaseManager: *modulebase.NewBaseManager("webconsole", "", "", nil, nil),
		Keyword: "webconsole", KeywordPlural: "webconsole"}
}

func (m WebConsoleManager) DoConnect(s *mcclient.ClientSession, connType, id, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(connType) == 0 {
		return nil, fmt.Errorf("Empty connection resource type")
	}
	url := fmt.Sprintf("/webconsole/%s", connType)
	if id != "" {
		url = fmt.Sprintf("%s/%s", url, id)
	}
	if action != "" {
		url = fmt.Sprintf("%s/%s", url, action)
	}
	return modulebase.Post(m.ResourceManager, s, url, params, "webconsole")
}

func (m WebConsoleManager) DoK8sConnect(s *mcclient.ClientSession, id, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "k8s", id, action, params)
}

func (m WebConsoleManager) DoK8sShellConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoK8sConnect(s, id, "shell", params)
}

func (m WebConsoleManager) DoK8sLogConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoK8sConnect(s, id, "log", params)
}

func (m WebConsoleManager) DoBaremetalConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "baremetal", id, "", params)
}

func (m WebConsoleManager) DoSshConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "ssh", id, "", params)
}

func (m WebConsoleManager) DoServerConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "server", id, "", params)
}
