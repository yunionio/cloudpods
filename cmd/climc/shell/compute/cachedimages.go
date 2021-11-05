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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CachedImageListOptions struct {
		options.BaseListOptions
		ImageType string `help:"image type" choices:"system|customized|shared|market"`

		Region string `help:"show images cached at cloud region"`
		Zone   string `help:"show images cached at zone"`

		HostSchedtagId string `help:"filter cached image with host schedtag"`
		Valid          *bool  `help:"valid cachedimage"`
	}
	R(&CachedImageListOptions{}, "cached-image-list", "List cached images", func(s *mcclient.ClientSession, args *CachedImageListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Cachedimages.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cachedimages.GetColumns(s))
		return nil
	})

	type CachedImageShowOptions struct {
		ID string `help:"ID or Name of the cached image to show"`
	}
	R(&CachedImageShowOptions{}, "cached-image-show", "Show cached image details", func(s *mcclient.ClientSession, args *CachedImageShowOptions) error {
		result, err := modules.Cachedimages.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CachedImageShowOptions{}, "cached-image-refresh", "Refresh cached image details", func(s *mcclient.ClientSession, args *CachedImageShowOptions) error {
		result, err := modules.Cachedimages.PerformAction(s, args.ID, "refresh", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CachedImageDeleteOptions struct {
		ID []string `help:"ID or Name of the cached image to show"`
	}
	R(&CachedImageDeleteOptions{}, "cached-image-delete", "Remove cached image information", func(s *mcclient.ClientSession, args *CachedImageDeleteOptions) error {
		results := modules.Cachedimages.BatchDelete(s, args.ID, nil)
		printBatchResults(results, modules.Cachedimages.GetColumns(s))
		return nil
	})
}
