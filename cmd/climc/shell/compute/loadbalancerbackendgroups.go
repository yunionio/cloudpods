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
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	R(&options.LoadbalancerBackendGroupCreateOptions{}, "lbbackendgroup-create", "Create lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupGetOptions{}, "lbbackendgroup-show", "Show lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupGetOptions) error {
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupListOptions{}, "lbbackendgroup-list", "List lbbackendgroups", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupListOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerBackendGroups.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerBackendGroups.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerBackendGroupUpdateOptions{}, "lbbackendgroup-update", "Update lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupUpdateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupDeleteOptions{}, "lbbackendgroup-delete", "Delete lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupDeleteOptions) error {
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupDeleteOptions{}, "lbbackendgroup-purge", "Purge lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupDeleteOptions) error {
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})

}
