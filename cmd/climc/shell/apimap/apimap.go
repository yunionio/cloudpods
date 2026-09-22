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

package apimap

import (
	"fmt"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/apimap"
)

func init() {
	type GetOptions struct{}
	shell.R(new(GetOptions), "apimap-modelsets", "Show the model sets served by the apimap service", func(s *mcclient.ClientSession, _ *GetOptions) error {
		ret, _, err := apimap.APIMap.GetModelSets(s)
		if err != nil {
			return err
		}
		fmt.Print(ret.YAMLString())
		return nil
	})

	type StatsOptions struct{}
	shell.R(new(StatsOptions), "apimap-modelsets-stats", "Show the sync status of the apimap service", func(s *mcclient.ClientSession, _ *StatsOptions) error {
		ret, err := apimap.APIMap.GetModelSetsStats(s)
		if err != nil {
			return err
		}
		fmt.Print(ret.YAMLString())
		return nil
	})
}
