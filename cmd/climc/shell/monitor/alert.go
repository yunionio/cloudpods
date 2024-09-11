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
	"fmt"

	"yunion.io/x/pkg/errors"

	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := NewResourceCmd(monitor.Alerts)
	cmd.List(new(options.AlertListOptions))
	// cmd.Create(new(options.AlertCreateOptions))
	cmd.Show(new(options.AlertShowOptions))
	cmd.Update(new(options.AlertUpdateOptions))
	cmd.BatchDelete(new(options.AlertDeleteOptions))
	cmd.Perform("pause", new(options.AlertPauseOptions))

	aN := cmdN("alert")
	R(&options.AlertTestRunOptions{}, aN("test-run"), "Test run alert",
		func(s *mcclient.ClientSession, args *options.AlertTestRunOptions) error {
			data := new(monitorapi.AlertTestRunInput)
			if args.Debug {
				data.IsDebug = true
			}
			ret, err := monitor.Alerts.DoTestRun(s, args.ID, data)
			if err != nil {
				return errors.Wrap(err, "DoTestRun")
			}
			fmt.Printf("%s\n", ret.PrettyString())
			return nil
		})
}
