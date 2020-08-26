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
	type SubscriptionListOptions struct {
	}
	shellutils.R(&SubscriptionListOptions{}, "subscription-list", "List subscriptions", func(cli *azure.SRegion, args *SubscriptionListOptions) error {
		subscriptions, err := cli.GetClient().GetSubscriptions()
		if err != nil {
			return err
		}
		printList(subscriptions, 0, 0, 0, nil)
		return nil
	})

	type SubscriptionCreateOptions struct {
		NAME      string
		OfferType string `choices:"MS-AZR-0017P|MS-AZR-0148P" default:"MS-AZR-0017P"`
		EAID      string `help:"Enrollment accounts id"`
	}

	shellutils.R(&SubscriptionCreateOptions{}, "subscription-create", "Create subscription", func(cli *azure.SRegion, args *SubscriptionCreateOptions) error {
		return cli.GetClient().CreateSubscription(args.NAME, args.EAID, args.OfferType)
	})

	type ServicePrincipalListOptions struct {
		AppId string
	}

	shellutils.R(&ServicePrincipalListOptions{}, "sp-list", "List service principal", func(cli *azure.SRegion, args *ServicePrincipalListOptions) error {
		sp, err := cli.GetClient().ListServicePrincipal(args.AppId)
		if err != nil {
			return err
		}
		printList(sp, 0, 0, 0, nil)
		return nil
	})

}
