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

import (
	"os"

	"yunion.io/x/jsonutils"
)

type SDocker struct {
	*SBaseSystemService
}

func NewDockerService() *SDocker {
	return &SDocker{NewBaseSystemService("docker", nil)}
}

func (s *SDocker) GetConfig(kwargs map[string]interface{}) string {
	return jsonutils.Marshal(kwargs).PrettyString()
}

func (s *SDocker) GetConfigFile() string {
	os.MkdirAll("/etc/docker", 0755)
	return "/etc/docker/daemon.json"
}

func (s *SDocker) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SDocker) BgReload(kwargs map[string]interface{}) {
	go s.Reload(kwargs)
}

func (s *SDocker) BgReloadConf(kwargs map[string]interface{}) {
	go s.reloadConf(s.GetConfig(kwargs), s.GetConfigFile())
}
