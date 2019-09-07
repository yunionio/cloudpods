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
	type StorageCachedImageListOptions struct {
		options.BaseListOptions
		Storagecache string `help:"ID or Name of Storage"`
		Image        string `help:"ID or Name of image"`
	}
	R(&StorageCachedImageListOptions{}, "storage-cached-image-list", "List storage cached image pairs", func(s *mcclient.ClientSession, args *StorageCachedImageListOptions) error {
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
		if len(args.Storagecache) > 0 {
			result, err = modules.Storagecachedimages.ListDescendent(s, args.Storagecache, params)
		} else if len(args.Image) > 0 {
			result, err = modules.Storagecachedimages.ListDescendent2(s, args.Image, params)
		} else {
			result, err = modules.Storagecachedimages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Storagecachedimages.GetColumns(s))
		return nil
	})

	type StorageCachedImageUpdateOptions struct {
		STORAGECACHE string `help:"ID or Name of Storage cache"`
		IMAGE        string `help:"ID or name of image"`
		Status       string `help:"Status"`
	}
	R(&StorageCachedImageUpdateOptions{}, "storage-cached-image-update", "Update storage cached image", func(s *mcclient.ClientSession, args *StorageCachedImageUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Status) > 0 {
			params.Add(jsonutils.NewString(args.Status), "status")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Storagecachedimages.Update(s, args.STORAGECACHE, args.IMAGE, nil, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type StorageCachedImageShowOptions struct {
		STORAGECACHE string `help:"ID or Name of Storage cache"`
		IMAGE        string `help:"ID or name of image"`
	}
	R(&StorageCachedImageShowOptions{}, "storage-cached-image-show", "Show storage cached image", func(s *mcclient.ClientSession, args *StorageCachedImageShowOptions) error {
		result, err := modules.Storagecachedimages.Get(s, args.STORAGECACHE, args.IMAGE, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
