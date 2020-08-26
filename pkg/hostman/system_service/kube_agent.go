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

package system_service

import "fmt"

type SKubeAgent struct {
	*SBaseSystemService
}

func NewKubeAgentService() *SKubeAgent {
	return &SKubeAgent{NewBaseSystemService("yunion-kube-agent", nil)}
}

func (s *SKubeAgent) GetConfig(kwargs map[string]interface{}) string {
	conf := ""
	conf += fmt.Sprintf("server = \"%s\"\n", kwargs["serverUrl"])
	conf += fmt.Sprintf("token = \"%s\"\n", kwargs["token"])
	conf += fmt.Sprintf("id = \"%s\"\n", kwargs["nodeId"])
	return conf
}

func (s *SKubeAgent) GetConfigFile() string {
	return "/etc/yunion/kube-agent.conf"
}

func (s *SKubeAgent) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SKubeAgent) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SKubeAgent) BgReloadConf(kwargs map[string]interface{}) {
	go s.reloadConf(s.GetConfig(kwargs), s.GetConfigFile())
}
