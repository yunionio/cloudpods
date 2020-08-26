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

package oldmodels

type SConfigManager struct {
	SStatusStandaloneResourceBaseManager
}

var ConfigManager *SConfigManager

func init() {
	ConfigManager = &SConfigManager{
		SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(
			SConfig{},
			"notify_t_config",
			"oldconfig",
			"oldconfigs",
		),
	}
	ConfigManager.SetVirtualObject(ConfigManager)
}

// SConfig is a table which storage (k,v) and its type.
// The three important concepts are key, value and type.
// Key and type uniquely identify a value.
type SConfig struct {
	SStatusStandaloneResourceBase

	Type      string `width:"15" nullable:"false" create:"required" list:"user"`
	KeyText   string `width:"30" nullable:"false" create:"required" list:"user"`
	ValueText string `width:"256" nullable:"false" create:"required" list:"user"`
}
