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
	R(&options.MetadataListOptions{}, "metadata-list", "List metadatas", func(s *mcclient.ClientSession, opts *options.MetadataListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Metadatas.List(s, params)
		if err != nil {
			return err
		}
		printList(result, []string{})
		return nil
	})

	R(&options.TagListOptions{}, "tag-list", "List tags", func(s *mcclient.ClientSession, opts *options.TagListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Metadatas.Get(s, "tag-value-pairs", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
