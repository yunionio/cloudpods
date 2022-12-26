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
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.MetadataListOptions{}, "metadata-list", "List metadatas", func(s *mcclient.ClientSession, opts *options.MetadataListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.ComputeMetadatas.List(s, params)
		if err != nil {
			return err
		}
		printList(result, []string{})
		return nil
	})

	R(&options.TagListOptions{}, "tag-list", "List tags", func(s *mcclient.ClientSession, opts *options.TagListOptions) error {
		var mod modulebase.IResourceManager
		switch opts.Service {
		case "compute":
			mod = &modules.ComputeMetadatas
		case "identity":
			mod = &modules.IdentityMetadatas
		case "image":
			mod = &modules.ImageMetadatas
		default:
			mod = &modules.ComputeMetadatas
		}
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := mod.Get(s, "tag-value-pairs", params)
		if err != nil {
			return err
		}
		listResult := printutils.ListResult{}
		err = result.Unmarshal(&listResult)
		if err != nil {
			return err
		}
		printList(&listResult, []string{})
		return nil
	})

}
