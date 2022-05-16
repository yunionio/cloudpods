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

package monitor

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	MigrationAlertManager *SMigrationAlertManager
)

func init() {
	MigrationAlertManager = NewMigrationAlertManager()
	m := MigrationAlertManager
	modules.Register(m)
	m.SetApiVersion(mcclient.V2_API_VERSION)
	modules.Register(m)
}

type SMigrationAlertManager struct {
	*modulebase.ResourceManager
}

func NewMigrationAlertManager() *SMigrationAlertManager {
	man := modules.NewMonitorV2Manager("migrationalert", "migrationalerts",
		[]string{"id", "name", "metric_type"},
		[]string{})
	return &SMigrationAlertManager{
		ResourceManager: &man,
	}
}
