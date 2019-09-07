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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type HostCachedImageListOptions struct {
		options.BaseListOptions
		Host  string `help:"ID or Name of Host"`
		Image string `help:"ID or Name of image"`
	}
	R(&HostCachedImageListOptions{}, "host-cachedimage-list", "List host cached image pairs", func(s *mcclient.ClientSession, args *HostCachedImageListOptions) error {
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
			result, err = modules.Hostcachedimages.ListDescendent(s, args.Host, params)
		} else {
			result, err = modules.Hostcachedimages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Hostcachedimages.GetColumns(s))
		return nil
	})

	type HostCachedImageUpdateOptions struct {
		HOST   string `help:"ID or Name of Host"`
		IMAGE  string `help:"ID or name of image"`
		Status string `help:"Status"`
	}
	R(&HostCachedImageUpdateOptions{}, "host-cachedimage-update", "Update host cached image", func(s *mcclient.ClientSession, args *HostCachedImageUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Status) > 0 {
			params.Add(jsonutils.NewString(args.Status), "status")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Hostcachedimages.Update(s, args.HOST, args.IMAGE, nil, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
