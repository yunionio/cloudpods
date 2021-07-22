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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type BudgetListOptions struct {
	options.BaseListOptions
}

func (opt *BudgetListOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type BudgetCreateOptions struct {
	Name       string
	PeriodType string `help:"period of budget, example:year/quarter/month/day" json:"period_type"`
	StartTime  string `help:"start time of budget" json:"start_time"`
	EndTime    string `help:"end time of budget" json:"end_time"`

	Brand             string `help:"brand of budget cost filter" json:"brand"`
	CloudaccountId    string `help:"cloudaccount id of budget cost filter" json:"cloudaccount_id"`
	Cloudaccount      string `help:"cloudaccount name of budget cost filter" json:"cloudaccount"`
	CloudproviderId   string `help:"cloudprovider id of budget cost filter" json:"cloudprovider_id"`
	CloudproviderName string `help:"cloudprovider name of budget cost filter" json:"cloudprovider_name"`
	RegionId          string `help:"region id of budget cost filter" json:"region_id"`
	Region            string `help:"region name of budget cost filter" json:"region"`
	DomainIdFilter    string `help:"domain id of budget cost filter" json:"domain_id_filter"`
	ProjectIdFilter   string `help:"project id of budget cost filter" json:"project_id_filter"`
	DomainFilter      string `help:"domain of budget cost filter" json:"domain_filter"`
	ProjectFilter     string `help:"project of budget cost filter" json:"project_filter"`
	ResourceType      string `help:"resource type of budget cost filter" json:"resource_type"`
	Currency          string `help:"currency of budget cost filter" json:"currency"`

	Amount float64 `help:"amount of budget cost" json:"amount"`
}

func (opt *BudgetCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

type BudgetUpdateOptions struct {
	ID string `help:"ID of budget" json:"-"`

	Brand             string `help:"brand of budget cost filter" json:"brand"`
	CloudaccountId    string `help:"cloudaccount id of budget cost filter" json:"cloudaccount_id"`
	Cloudaccount      string `help:"cloudaccount name of budget cost filter" json:"cloudaccount"`
	CloudproviderId   string `help:"cloudprovider id of budget cost filter" json:"cloudprovider_id"`
	CloudproviderName string `help:"cloudprovider name of budget cost filter" json:"cloudprovider_name"`
	RegionId          string `help:"region id of budget cost filter" json:"region_id"`
	Region            string `help:"region name of budget cost filter" json:"region"`
	DomainIdFilter    string `help:"domain id of budget cost filter" json:"domain_id_filter"`
	ProjectIdFilter   string `help:"project id of budget cost filter" json:"project_id_filter"`
	DomainFilter      string `help:"domain of budget cost filter" json:"domain_filter"`
	ProjectFilter     string `help:"project of budget cost filter" json:"project_filter"`
	ResourceType      string `help:"resource type of budget cost filter" json:"resource_type"`
	Currency          string `help:"currency of budget cost filter" json:"currency"`

	Amount float64 `help:"amount of budget cost" json:"amount"`
}

func (opt *BudgetUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *BudgetUpdateOptions) GetId() string {
	return opt.ID
}

type BudgetDeleteOptions struct {
	ID string `help:"ID of budget" json:"-"`
}

func (opt *BudgetDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *BudgetDeleteOptions) GetId() string {
	return opt.ID
}

type BudgetSyncAlertsOptions struct {
	ID string `help:"ID of budget" json:"-"`

	Alerts []BudgetAlert `help:"alerts of budget" json:"alerts"`
}

type BudgetAlert struct {
	AlertType string `help:"alert type, example:cost" json:"alert_type"`

	AlertThresholdType string   `help:"alert type, example:percert/amount" json:"alert_threshold_type"`
	AlertThreshold     float64  `help:"alert amount" json:"alert_threshold"`
	UserIds            []string `help:"id of users" json:"user_ids"`
}

func (opt *BudgetSyncAlertsOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (opt *BudgetSyncAlertsOptions) GetId() string {
	return opt.ID
}
