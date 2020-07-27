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

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initChart() {
	cmdN := func(suffix string) string {
		return resourceCmdN("chart", suffix)
	}

	R(&o.ChartListOptions{}, cmdN("list"), "List k8s helm global charts", func(s *mcclient.ClientSession, args *o.ChartListOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		charts, err := k8s.Charts.List(s, params)
		if err != nil {
			return err
		}

		PrintListResultTable(charts, k8s.Charts, s)
		return nil
	})

	R(&o.ChartGetOptions{}, cmdN("show"), "Show details of a chart", func(s *mcclient.ClientSession, args *o.ChartGetOptions) error {
		chart, err := k8s.Charts.Get(s, args.NAME, args.Params())
		if err != nil {
			return err
		}
		printObjectYAML(chart)
		return nil
	})
}
