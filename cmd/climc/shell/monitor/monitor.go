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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func cmdN(suffix string) func(action string) string {
	return func(action string) string {
		return fmt.Sprintf("monitor-%s-%s", suffix, action)
	}
}

func init() {
	dsN := cmdN("datasource")
	R(&options.DataSourceListOptions{}, dsN("list"), "List all monitor data source",
		func(s *mcclient.ClientSession, args *options.DataSourceListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.DataSources.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, monitor.DataSources.GetColumns(s))
			return nil
		})

	R(&options.DataSourceDeleteOptions{}, dsN("delete"), "Delete monitor data source",
		func(s *mcclient.ClientSession, args *options.DataSourceDeleteOptions) error {
			ret, err := monitor.DataSources.Delete(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}
