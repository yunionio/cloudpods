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

package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type BaremetalStorageListOptions struct {
		options.BaseListOptions
		Host string `help:"ID or Name of Host"`
	}
	R(&BaremetalStorageListOptions{}, "baremetal-storage-list", "List baremetal storage pairs", func(s *mcclient.ClientSession, args *BaremetalStorageListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		var result *modulebase.ListResult
		var err error
		if len(args.Host) > 0 {
			result, err = modules.Baremetalstorages.ListDescendent(s, args.Host, params)
		} else {
			result, err = modules.Baremetalstorages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Baremetalstorages.GetColumns(s))
		return nil
	})

	type BaremetalStorageDetailOptions struct {
		HOST    string `help:"ID or Name of Host"`
		STORAGE string `help:"ID or Name of Storage"`
	}
	R(&BaremetalStorageDetailOptions{}, "baremetal-storage-show", "Show baremetal storage details", func(s *mcclient.ClientSession, args *BaremetalStorageDetailOptions) error {
		result, err := modules.Baremetalstorages.Get(s, args.HOST, args.STORAGE, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
