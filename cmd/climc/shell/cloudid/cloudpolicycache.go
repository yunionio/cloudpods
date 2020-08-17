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

package cloudid

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CloudpolicycacheListOptions struct {
		options.BaseListOptions

		CloudpolyId    string
		CloudaccountId string
	}
	R(&CloudpolicycacheListOptions{}, "cloud-policy-cache-list", "List cloud policy caches", func(s *mcclient.ClientSession, opts *CloudpolicycacheListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Cloudpolicycaches.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudpolicycaches.GetColumns(s))
		return nil
	})
}
