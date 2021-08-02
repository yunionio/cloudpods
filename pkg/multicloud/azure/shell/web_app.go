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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AppSiteListOptions struct {
	}
	shellutils.R(&AppSiteListOptions{}, "app-site-list", "List app service plan", func(cli *azure.SRegion, args *AppSiteListOptions) error {
		ass, err := cli.GetAppSites()
		if err != nil {
			return err
		}
		printList(ass, len(ass), 0, 0, []string{})
		return nil
	})
	type AppSiteShowOptions struct {
		ID string
	}
	shellutils.R(&AppSiteShowOptions{}, "app-site-show", "Show app service plan", func(cli *azure.SRegion, args *AppSiteShowOptions) error {
		as, err := cli.GetAppSite(args.ID)
		if err != nil {
			return err
		}
		printObject(as)
		return nil
	})
	shellutils.R(&AppSiteShowOptions{}, "app-site-deployment-list", "List sku usable in app service plan", func(cli *azure.SRegion, args *AppSiteShowOptions) error {
		as, err := cli.GetAppSite(args.ID)
		if err != nil {
			return err
		}
		deployments, err := as.GetDeployments()
		if err != nil {
			return err
		}
		printList(deployments, len(deployments), 0, 0, []string{})
		return nil
	})
	shellutils.R(&AppSiteShowOptions{}, "app-site-slot-list", "List slots ofr App site", func(cli *azure.SRegion, args *AppSiteShowOptions) error {
		as, err := cli.GetAppSite(args.ID)
		if err != nil {
			return err
		}
		slots, err := as.GetSlots()
		if err != nil {
			return err
		}
		printList(slots, len(slots), 0, 0, []string{})
		return nil
	})
}
