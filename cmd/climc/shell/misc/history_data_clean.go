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

package misc

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type HistoryDataCleanOptions struct {
		SERVICE string `help:"Service type"`
		Day     *int   `default:"30"`
	}
	R(&HistoryDataCleanOptions{}, "history-data-clean", "clean history data", func(s *mcclient.ClientSession, args *HistoryDataCleanOptions) error {
		body, err := modules.HistoryDataClean(s, args.SERVICE, args.Day)
		if err != nil {
			return err
		}
		fmt.Println(body.PrettyString())
		return nil
	})
}
