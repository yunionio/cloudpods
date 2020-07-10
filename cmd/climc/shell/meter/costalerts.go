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

package meter

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CostAlertListOptions struct {
		options.BaseListOptions
	}
	R(&CostAlertListOptions{}, "cost-alert-list", "list cost alerts",
		func(s *mcclient.ClientSession, args *CostAlertListOptions) error {
			params, err := options.ListStructToParams(args)
			if err != nil {
				return err
			}
			result, err := modules.CostAlerts.List(s, params)
			if err != nil {
				return err
			}
			printList(result, modules.CostAlerts.GetColumns(s))
			return nil
		})

	type CostAlertCreateOptions struct {
		Brand             string `help:"brand of cost filter" json:"brand"`
		AccountId         string `help:"cloudaccount id of cost filter" json:"account_id"`
		Account           string `help:"cloudaccount name of cost filter" json:"account"`
		CloudproviderId   string `help:"cloudprovider id of cost filter" json:"cloudprovider_id"`
		CloudproviderName string `help:"cloudprovider name of cost filter" json:"cloudprovider_name"`

		RegionId string `help:"region id of cost filter" json:"region_id"`
		Region   string `help:"region name of cost filter" json:"region"`

		DomainIdFilter  string `help:"domain id of cost filter" json:"domain_id_filter"`
		ProjectIdFilter string `help:"project id of cost filter" json:"project_id_filter"`
		DomainFilter    string `help:"domain of cost filter" json:"domain_filter"`
		ProjectFilter   string `help:"project of cost filter" json:"project_filter"`

		ResourceType string `help:"resource type of cost filter" json:"resource_type"`
		CostType     string `help:"cost type, example:month/day" json:"cost_type"`
		Currency     string `help:"currency of cost filter" json:"currency"`

		Amount float64 `help:"cost amount threshold" json:"amount"`

		UserIds string `help:"user ids of notifications, example:id1,id2" json:"user_ids"`
	}
	R(&CostAlertCreateOptions{}, "cost-alert-create", "create cost alert", func(s *mcclient.ClientSession, args *CostAlertCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		if len(args.UserIds) > 0 {
			userIds := strings.Split(args.UserIds, ",")
			item := jsonutils.NewDict()
			item.Set("content", jsonutils.NewStringArray(userIds))
			params.Set("user_ids", item)
		}

		result, err := modules.CostAlerts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CostAlertUpdateOptions struct {
		ID string `help:"ID of cost alert" json:"-"`

		Brand             string `help:"brand of cost filter" json:"brand"`
		AccountId         string `help:"cloudaccount id of cost filter" json:"account_id"`
		Account           string `help:"cloudaccount name of cost filter" json:"account"`
		CloudproviderId   string `help:"cloudprovider id of cost filter" json:"cloudprovider_id"`
		CloudproviderName string `help:"cloudprovider name of cost filter" json:"cloudprovider_name"`

		RegionId string `help:"region id of cost filter" json:"region_id"`
		Region   string `help:"region name of cost filter" json:"region"`

		DomainIdFilter  string `help:"domain id of cost filter" json:"domain_id_filter"`
		ProjectIdFilter string `help:"project id of cost filter" json:"project_id_filter"`
		DomainFilter    string `help:"domain of cost filter" json:"domain_filter"`
		ProjectFilter   string `help:"project of cost filter" json:"project_filter"`

		ResourceType string `help:"resource type of cost filter" json:"resource_type"`
		CostType     string `help:"cost type, example:month/day" json:"cost_type"`
		Currency     string `help:"currency of cost filter" json:"currency"`

		Amount float64 `help:"cost amount threshold" json:"amount"`

		UserIds string `help:"user ids of notifications, example:id1,id2" json:"user_ids"`
	}
	R(&CostAlertUpdateOptions{}, "cost-alert-update", "update cost alert", func(s *mcclient.ClientSession, args *CostAlertUpdateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		if len(args.UserIds) > 0 {
			userIds := strings.Split(args.UserIds, ",")
			item := jsonutils.NewDict()
			item.Set("content", jsonutils.NewStringArray(userIds))
			params.Set("user_ids", item)
		}

		result, err := modules.CostAlerts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CostAlertDeleteOptions struct {
		ID string `help:"ID of cost alert" json:"-"`
	}
	R(&CostAlertDeleteOptions{}, "cost-alert-delete", "delete cost alert", func(s *mcclient.ClientSession, args *CostAlertDeleteOptions) error {
		result, err := modules.CostAlerts.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
