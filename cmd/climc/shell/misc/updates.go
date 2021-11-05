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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/autoupdate"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SUpdateListOptions struct {
		options.BaseListOptions
		Region string `help:"cloud region ID or Name"`
	}

	R(&SUpdateListOptions{}, "update-list", "List updates", func(s *mcclient.ClientSession, args *SUpdateListOptions) error {
		// TODO filer by region
		result, err := modules.Updates.List(s, nil)

		if err != nil {
			return err
		}

		printList(result, modules.Updates.GetColumns(s))
		return nil
	})

	type SUpdatePerformOptions struct {
		Cmp     bool `help:"update Controller And all the Compute nodes automatically"`
		CmpOnly bool `help:"Updates only computes nodes, excluding controller nodes"`
	}

	R(&SUpdatePerformOptions{}, "update-perform", "Update the Controler", func(s *mcclient.ClientSession, args *SUpdatePerformOptions) error {
		params := jsonutils.NewDict()

		if args.Cmp && args.CmpOnly {
			return fmt.Errorf("--cmp and --cmp-only can't go together")
		}

		if args.Cmp {
			params.Add(jsonutils.JSONTrue, "cmp")
		} else if args.CmpOnly {
			params.Add(jsonutils.JSONTrue, "cmp_only")
		}

		modules.Updates.PerformAction(s, "", "", params)

		result, err := modules.Updates.List(s, nil)
		if err != nil {
			return err
		}
		printList(result, modules.Updates.GetColumns(s))
		return nil
	})
}
