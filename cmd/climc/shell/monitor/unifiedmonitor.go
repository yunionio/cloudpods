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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(monitor.UnifiedMonitorManager).SetPrefix("monitor")
	cmd.Show(&options.SimpleQueryOptions{})
	cmd.GetProperty(&options.MeasurementsQueryOptions{})
	// cmd.GetProperty(&options.DatabasesQueryOptions{})

	R(new(options.MetricQueryOptions), "monitor-unifiedmonitor-query", "Perform metrics query", func(s *mcclient.ClientSession, opts *options.MetricQueryOptions) error {
		input, err := opts.GetQueryInput()
		if err != nil {
			return err
		}
		resp, err := monitor.UnifiedMonitorManager.PerformQuery(s, input)
		if err != nil {
			return errors.Wrap(err, "PerformQuery")
		}
		printObject(resp)
		return nil
	})
}
