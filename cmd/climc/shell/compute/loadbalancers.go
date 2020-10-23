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
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.LoadbalancerCreateOptions{}, "lb-create", "Create lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lb, err := modules.Loadbalancers.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerGetOptions{}, "lb-show", "Show lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerGetOptions) error {
		lb, err := modules.Loadbalancers.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerListOptions{}, "lb-list", "List lbs", func(s *mcclient.ClientSession, opts *options.LoadbalancerListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Loadbalancers.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Loadbalancers.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerUpdateOptions{}, "lb-update", "Update lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerUpdateOptions) error {
		params, err := options.StructToParams(opts)
		lb, err := modules.Loadbalancers.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerDeleteOptions{}, "lb-delete", "Delete lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerDeleteOptions) error {
		lb, err := modules.Loadbalancers.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerPurgeOptions{}, "lb-purge", "Purge lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerPurgeOptions) error {
		lb, err := modules.Loadbalancers.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerActionStatusOptions{}, "lb-status", "Change lb status", func(s *mcclient.ClientSession, opts *options.LoadbalancerActionStatusOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lb, err := modules.Loadbalancers.PerformAction(s, opts.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerActionSyncStatusOptions{}, "lb-syncstatus", "Sync lb status", func(s *mcclient.ClientSession, opts *options.LoadbalancerActionSyncStatusOptions) error {
		lb, err := modules.Loadbalancers.PerformAction(s, opts.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})

	R(&options.LoadbalancerGetOptions{}, "lb-change-owner-candidate-domains", "Get change owner candidate domain list", func(s *mcclient.ClientSession, args *options.LoadbalancerGetOptions) error {
		result, err := modules.Loadbalancers.GetSpecific(s, args.ID, "change-owner-candidate-domains", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ResourceMetadataOptions{}, "lb-add-tag", "Set tag of a lb", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Loadbalancers.PerformAction(s, opts.ID, "user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ResourceMetadataOptions{}, "lb-set-tag", "Set tag of a lb", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Loadbalancers.PerformAction(s, opts.ID, "set-user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.LoadbalancerRemoteUpdateOptions{}, "lb-remote-update", "Change lb status", func(s *mcclient.ClientSession, opts *options.ServerRemoteUpdateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lb, err := modules.Loadbalancers.PerformAction(s, opts.ID, "remote-update", params)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})

}
