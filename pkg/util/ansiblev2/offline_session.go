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

package ansiblev2

import (
	"context"
	"io"
)

type OfflineSession struct {
	PlaybookSessionBase

	playbookPath string
	proxy        string
	user         string
	hostIp       string
	hostName     string
	configs      map[string]interface{}
	configYaml   string
}

func NewOfflineSession() *OfflineSession {
	sess := &OfflineSession{
		PlaybookSessionBase: NewPlaybookSessionBase(),
	}
	return sess
}

func (sess *OfflineSession) PrivateKey(s string) *OfflineSession {
	sess.privateKey = s
	return sess
}

func (sess *OfflineSession) PlaybookPath(s string) *OfflineSession {
	sess.playbookPath = s
	return sess
}

func (sess *OfflineSession) Inventory(s string) *OfflineSession {
	sess.inventory = s
	return sess
}

func (sess *OfflineSession) OutputWriter(w io.Writer) *OfflineSession {
	sess.outputWriter = w
	return sess
}

func (sess *OfflineSession) KeepTmpdir(keep bool) *OfflineSession {
	sess.keepTmpdir = keep
	return sess
}

func (sess *OfflineSession) Configs(configs map[string]interface{}) *OfflineSession {
	sess.configs = configs
	return sess
}

func (sess *OfflineSession) ConfigYaml(yaml string) *OfflineSession {
	sess.configYaml = yaml
	return sess
}

func (sess *OfflineSession) GetPlaybookPath() string {
	return sess.playbookPath
}

func (sess *OfflineSession) GetConfigs() map[string]interface{} {
	return sess.configs
}

func (sess *OfflineSession) GetConfigYaml() string {
	return sess.configYaml
}

func (sess *OfflineSession) Run(ctx context.Context) (err error) {
	return runnable{sess}.Run(ctx)
}
