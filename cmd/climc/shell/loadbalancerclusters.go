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
	R(&options.LoadbalancerClusterCreateOptions{}, "lbcluster-create", "Create lbcluster", func(s *mcclient.ClientSession, opts *options.LoadbalancerClusterCreateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lbcluster, err := modules.LoadbalancerClusters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lbcluster)
		return nil
	})
	R(&options.LoadbalancerClusterUpdateOptions{}, "lbcluster-update", "Update lbcluster", func(s *mcclient.ClientSession, opts *options.LoadbalancerClusterUpdateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lbcluster, err := modules.LoadbalancerClusters.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lbcluster)
		return nil
	})
	R(&options.LoadbalancerClusterGetOptions{}, "lbcluster-show", "Show lbcluster", func(s *mcclient.ClientSession, opts *options.LoadbalancerClusterGetOptions) error {
		lbcluster, err := modules.LoadbalancerClusters.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbcluster)
		return nil
	})
	R(&options.LoadbalancerClusterListOptions{}, "lbcluster-list", "List lbclusters", func(s *mcclient.ClientSession, opts *options.LoadbalancerClusterListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerClusters.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerClusters.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerClusterDeleteOptions{}, "lbcluster-delete", "Show lbcluster", func(s *mcclient.ClientSession, opts *options.LoadbalancerClusterDeleteOptions) error {
		lbagent, err := modules.LoadbalancerClusters.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbagent)
		return nil
	})
}
