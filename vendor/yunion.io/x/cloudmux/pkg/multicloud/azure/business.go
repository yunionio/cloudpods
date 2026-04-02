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

package azure

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
)

type SMonthBill struct {
	Month         string                 `json:"month"`
	StartDate     string                 `json:"start_date"`
	EndDate       string                 `json:"end_date"`
	Currency      string                 `json:"currency"`
	Total         float64                `json:"total"`
	Metric        string                 `json:"metric"`
	Granularity   string                 `json:"granularity"`
	Subscriptions []SMonthBillServiceFee `json:"subscriptions"`
}

type SMonthBillServiceFee struct {
	SubscriptionId string  `json:"subscription_id"`
	Amount         float64 `json:"amount"`
	Unit           string  `json:"unit"`
}

type SCostQueryResult struct {
	Properties struct {
		Columns []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"columns"`
		Rows [][]interface{} `json:"rows"`
	} `json:"properties"`
}

func getMonthDateRange(month string) (string, string, error) {
	monthStart, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", errors.Wrapf(err, "invalid month %q, expected YYYY-MM", month)
	}
	monthEnd := monthStart.AddDate(0, 1, -1)
	return monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"), nil
}

func (self *SAzureClient) GetMonthBill(month string) (*SMonthBill, error) {
	startDate, endDate, err := getMonthDateRange(month)
	if err != nil {
		return nil, err
	}
	billingAccounts, err := self.GetBillingAccounts()
	if err != nil {
		return nil, errors.Wrap(err, "GetBillingAccounts")
	}
	if len(billingAccounts) == 0 {
		return nil, errors.Errorf("no available billing accounts")
	}

	ret := &SMonthBill{
		Month:         month,
		StartDate:     startDate,
		EndDate:       endDate,
		Metric:        "PreTaxCost",
		Granularity:   "monthly",
		Subscriptions: make([]SMonthBillServiceFee, 0),
	}
	feeBySubscriptionId := map[string]float64{}
	for i := range billingAccounts {
		accountPath := strings.TrimSpace(billingAccounts[i].Id)
		if len(accountPath) == 0 && len(billingAccounts[i].Name) > 0 {
			accountPath = fmt.Sprintf("/providers/Microsoft.Billing/billingAccounts/%s", billingAccounts[i].Name)
		}
		if len(accountPath) == 0 {
			continue
		}

		resource := fmt.Sprintf("%s/providers/Microsoft.CostManagement/query", strings.TrimSuffix(accountPath, "/"))
		resp, err := self.post_v2(resource, "2025-03-01", map[string]interface{}{
			"type":      "ActualCost",
			"timeframe": "Custom",
			"timePeriod": map[string]string{
				"from": fmt.Sprintf("%s", startDate),
				"to":   fmt.Sprintf("%s", endDate),
			},
			"dataset": map[string]interface{}{
				"granularity": "None",
				"aggregation": map[string]interface{}{
					"totalCost": map[string]string{
						"name":     "PreTaxCost",
						"function": "Sum",
					},
				},
				"grouping": []map[string]string{
					{
						"type": "Dimension",
						"name": "SubscriptionId",
					},
				},
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "post_v2 cost query for billing account %s", billingAccounts[i].Name)
		}

		queryRet := SCostQueryResult{}
		if err := resp.Unmarshal(&queryRet); err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal cost query for billing account %s", billingAccounts[i].Name)
		}

		subscriptionIdx, costIdx, currencyIdx := -1, -1, -1
		for c := range queryRet.Properties.Columns {
			name := strings.ToLower(queryRet.Properties.Columns[c].Name)
			switch name {
			case "subscriptionid", "subscriptionname":
				subscriptionIdx = c
			case "totalcost", "pretaxcost":
				costIdx = c
			case "currency":
				currencyIdx = c
			}
		}
		if subscriptionIdx < 0 || costIdx < 0 {
			continue
		}

		for j := range queryRet.Properties.Rows {
			row := queryRet.Properties.Rows[j]
			if len(row) <= costIdx || len(row) <= subscriptionIdx {
				continue
			}
			subscription := strings.TrimSpace(fmt.Sprintf("%v", row[subscriptionIdx]))
			if len(subscription) == 0 {
				subscription = "Unknown"
			}
			cost, ok := row[costIdx].(float64)
			if !ok {
				continue
			}
			feeBySubscriptionId[subscription] += cost
			ret.Total += cost
			if len(ret.Currency) == 0 {
				if currencyIdx >= 0 && len(row) > currencyIdx {
					ret.Currency = strings.TrimSpace(fmt.Sprintf("%v", row[currencyIdx]))
				}
			}
		}
	}

	for subscriptionId, amount := range feeBySubscriptionId {
		ret.Subscriptions = append(ret.Subscriptions, SMonthBillServiceFee{
			SubscriptionId: subscriptionId,
			Amount:         amount,
			Unit:           ret.Currency,
		})
	}

	sort.Slice(ret.Subscriptions, func(i, j int) bool {
		return ret.Subscriptions[i].Amount > ret.Subscriptions[j].Amount
	})

	return ret, nil
}
