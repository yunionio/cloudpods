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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SecGroupCacheListOptions struct {
		options.BaseListOptions
		Secgroup string `help:"Secgroup ID or Name"`
	}

	R(&SecGroupCacheListOptions{}, "secgroup-cache-list", "List security group caches", func(s *mcclient.ClientSession, args *SecGroupCacheListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.SecGroupCaches.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.SecGroupCaches.GetColumns(s))
		return nil
	})
	type SecGroupCacheIdOptions struct {
		ID string `help:"ID or Name or secgroup cache"`
	}
	R(&SecGroupCacheIdOptions{}, "secgroup-cache-show", "Show security group cache", func(s *mcclient.ClientSession, args *SecGroupCacheIdOptions) error {
		result, err := modules.SecGroupCaches.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&SecGroupCacheIdOptions{}, "secgroup-cache-delete", "Delete security group cache", func(s *mcclient.ClientSession, args *SecGroupCacheIdOptions) error {
		result, err := modules.SecGroupCaches.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
