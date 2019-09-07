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
	type HostStorageListOptions struct {
		options.BaseListOptions
		Host    string `help:"ID or Name of Host"`
		Storage string `help:"ID or Name of Storage"`
	}
	R(&HostStorageListOptions{}, "host-storage-list", "List host storage pairs", func(s *mcclient.ClientSession, args *HostStorageListOptions) error {
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
		// if len(args.Storage) > 0 {
		// 	params.Add(jsonutils.NewString(args.Storage), "storage")
		// }
		if len(args.Host) > 0 {
			result, err = modules.Hoststorages.ListDescendent(s, args.Host, params)
		} else if len(args.Storage) > 0 {
			result, err = modules.Hoststorages.ListDescendent2(s, args.Storage, params)
		} else {
			result, err = modules.Hoststorages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Hoststorages.GetColumns(s))
		return nil
	})

	type HostStorageDetailOptions struct {
		HOST    string `help:"ID or Name of Host"`
		STORAGE string `help:"ID or Name of Storage"`
	}
	R(&HostStorageDetailOptions{}, "host-storage-show", "Show host storage details", func(s *mcclient.ClientSession, args *HostStorageDetailOptions) error {
		result, err := modules.Hoststorages.Get(s, args.HOST, args.STORAGE, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostStorageDetailOptions{}, "host-storage-detach", "Detach a storage from a host", func(s *mcclient.ClientSession, args *HostStorageDetailOptions) error {
		result, err := modules.Hoststorages.Detach(s, args.HOST, args.STORAGE, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostStorageAttachOptions struct {
		HOST       string `help:"ID or Name of Host"`
		STORAGE    string `help:"ID or Name of Storage"`
		MountPoint string `help:"MountPoint of Storage"`
	}

	R(&HostStorageAttachOptions{}, "host-storage-attach", "Attach a storage to a host", func(s *mcclient.ClientSession, args *HostStorageAttachOptions) error {
		params := jsonutils.NewDict()
		if args.MountPoint != "" {
			params.Add(jsonutils.NewString(args.MountPoint), "mount_point")
		}
		result, err := modules.Hoststorages.Attach(s, args.HOST, args.STORAGE, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
