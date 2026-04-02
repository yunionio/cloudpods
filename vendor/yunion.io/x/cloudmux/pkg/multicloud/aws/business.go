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

package aws

import (
	"context"
	"sort"
	"strconv"
	"time"

	aws2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/pkg/errors"
)

type SMonthBill struct {
	Month       string                 `json:"month"`
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	Currency    string                 `json:"currency"`
	Total       float64                `json:"total"`
	Metric      string                 `json:"metric"`
	Granularity string                 `json:"granularity"`
	Services    []SMonthBillServiceFee `json:"services"`
}

type SMonthBillServiceFee struct {
	Service string `json:"service"`
	Amount  string `json:"amount"`
	Unit    string `json:"unit"`
}

func getMonthDateRange(month string) (string, string, error) {
	monthStart, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", errors.Wrapf(err, "invalid month %q, expected YYYY-MM", month)
	}
	monthEnd := monthStart.AddDate(0, 1, 0)
	return monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"), nil
}

func (self *SAwsClient) getCostExplorerRegion() string {
	if self.GetAccessEnv() == api.CLOUD_ACCESS_ENV_AWS_CHINA {
		return "cn-northwest-1"
	}
	return "us-east-1"
}

func (self *SAwsClient) GetMonthBill(month string) (*SMonthBill, error) {
	startDate, endDate, err := getMonthDateRange(month)
	if err != nil {
		return nil, err
	}

	cfg, err := self.getConfig(context.Background(), self.getCostExplorerRegion(), true)
	if err != nil {
		return nil, errors.Wrap(err, "getConfig")
	}

	ceCli := costexplorer.NewFromConfig(cfg)
	resp, err := ceCli.GetCostAndUsage(context.Background(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws2.String(startDate),
			End:   aws2.String(endDate),
		},
		Granularity: cetypes.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []cetypes.GroupDefinition{
			{
				Type: cetypes.GroupDefinitionTypeDimension,
				Key:  aws2.String("SERVICE"),
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetCostAndUsage")
	}

	ret := &SMonthBill{
		Month:       month,
		StartDate:   startDate,
		EndDate:     endDate,
		Metric:      "UnblendedCost",
		Granularity: string(cetypes.GranularityMonthly),
		Services:    make([]SMonthBillServiceFee, 0),
	}

	var total float64
	for _, byTime := range resp.ResultsByTime {
		for _, group := range byTime.Groups {
			if len(group.Keys) == 0 {
				continue
			}
			metric, ok := group.Metrics["UnblendedCost"]
			if !ok {
				continue
			}
			if len(ret.Currency) == 0 {
				ret.Currency = aws2.ToString(metric.Unit)
			}
			amount := aws2.ToString(metric.Amount)
			if len(amount) > 0 {
				if val, parseErr := strconv.ParseFloat(amount, 64); parseErr == nil {
					total += val
				}
			}
			ret.Services = append(ret.Services, SMonthBillServiceFee{
				Service: group.Keys[0],
				Amount:  amount,
				Unit:    aws2.ToString(metric.Unit),
			})
		}
	}
	ret.Total = total

	sort.Slice(ret.Services, func(i, j int) bool {
		iVal, iErr := strconv.ParseFloat(ret.Services[i].Amount, 64)
		jVal, jErr := strconv.ParseFloat(ret.Services[j].Amount, 64)
		if iErr != nil || jErr != nil {
			return ret.Services[i].Amount > ret.Services[j].Amount
		}
		return iVal > jVal
	})

	return ret, nil
}
