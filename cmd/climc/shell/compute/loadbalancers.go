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
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Loadbalancers).WithKeyword("lb")
	cmd.Create(&options.LoadbalancerCreateOptions{})
	cmd.Show(&options.LoadbalancerIdOptions{})
	cmd.List(&options.LoadbalancerListOptions{})
	cmd.Update(&options.LoadbalancerUpdateOptions{})
	cmd.Delete(&options.LoadbalancerIdOptions{})
	cmd.Perform("purge", &options.LoadbalancerIdOptions{})
	cmd.Perform("status", &options.LoadbalancerActionStatusOptions{})
	cmd.Perform("syncstatus", &options.LoadbalancerIdOptions{})
	cmd.Perform("remote-update", &options.LoadbalancerRemoteUpdateOptions{})
	cmd.Get("change-owner-candidate-domains", &options.LoadbalancerIdOptions{})

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

}
