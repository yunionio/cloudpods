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
	R(&options.LoadbalancerBackendCreateOptions{}, "lbbackend-create", "Create lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbbackend, err := modules.LoadbalancerBackends.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendGetOptions{}, "lbbackend-show", "Show lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGetOptions) error {
		lbbackend, err := modules.LoadbalancerBackends.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendListOptions{}, "lbbackend-list", "List lbbackends", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendListOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerBackends.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerBackends.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerBackendUpdateOptions{}, "lbbackend-update", "Update lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendUpdateOptions) error {
		params, err := opts.Params()
		lbbackend, err := modules.LoadbalancerBackends.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendDeleteOptions{}, "lbbackend-delete", "Delete lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendDeleteOptions) error {
		lbbackend, err := modules.LoadbalancerBackends.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendDeleteOptions{}, "lbbackend-purge", "Purge lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendDeleteOptions) error {
		lbbackend, err := modules.LoadbalancerBackends.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
}
