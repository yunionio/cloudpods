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
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := NewResourceCmd(modules.CommonAlerts)
	cmd.Create(new(options.CommonAlertCreateOptions))
	cmd.List(new(options.CommonAlertListOptions))
	cmd.Show(new(options.CommonAlertShowOptions))
	cmd.Perform("enable", &options.CommonAlertShowOptions{})
	cmd.Perform("disable", &options.CommonAlertShowOptions{})
	cmd.BatchDelete(new(options.CommonAlertDeleteOptions))
	cmd.Perform("config", &options.CommonAlertUpdateOptions{})
}
