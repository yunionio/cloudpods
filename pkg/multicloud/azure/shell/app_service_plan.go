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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AppServicePlanListOptions struct {
	}
	shellutils.R(&AppServicePlanListOptions{}, "app-service-plan-list", "List app service plan", func(cli *azure.SRegion, args *AppServicePlanListOptions) error {
		appServicePlans, err := cli.GetAppServicePlans()
		if err != nil {
			return err
		}
		fmt.Printf("sku: %s\n", jsonutils.Marshal(appServicePlans[0].Sku))
		printList(appServicePlans, len(appServicePlans), 0, 0, []string{})
		return nil
	})
	type AppServicePlanShowOptions struct {
		ID string
	}
	shellutils.R(&AppServicePlanShowOptions{}, "app-service-plan-show", "Show app service plan", func(cli *azure.SRegion, args *AppServicePlanShowOptions) error {
		appServicePlan, err := cli.GetAppServicePlan(args.ID)
		if err != nil {
			return err
		}
		printObject(appServicePlan)
		return nil
	})
	shellutils.R(&AppServicePlanListOptions{}, "autoscale-setting-list", "List autoscale setting in app service plan", func(cli *azure.SRegion, args *AppServicePlanListOptions) error {
		appServicePlans, err := cli.GetAutoscaleSettingResources()
		if err != nil {
			return err
		}
		printList(appServicePlans, len(appServicePlans), 0, 0, []string{})
		return nil
	})
	shellutils.R(&AppServicePlanShowOptions{}, "app-service-sku-list", "List sku usable in app service plan", func(cli *azure.SRegion, args *AppServicePlanShowOptions) error {
		appServicePlan, err := cli.GetAppServicePlan(args.ID)
		if err != nil {
			return err
		}
		skus, err := appServicePlan.GetSkus()
		if err != nil {
			return err
		}
		printList(skus, len(skus), 0, 0, []string{})
		return nil
	})
	shellutils.R(&AppServicePlanShowOptions{}, "app-service-network-list", "List network connected with app service plan", func(cli *azure.SRegion, args *AppServicePlanShowOptions) error {
		appServicePlan, err := cli.GetAppServicePlan(args.ID)
		if err != nil {
			return err
		}
		networks, err := appServicePlan.GetNetworks()
		if err != nil {
			return err
		}
		printList(networks, len(networks), 0, 0, []string{})
		return nil
	})
}
