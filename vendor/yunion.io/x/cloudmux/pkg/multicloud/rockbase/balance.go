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

package rockbase

import (
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// SBalance 账户余额信息，对应 UBill GetBalance 接口。
type SBalance struct {
	Amount          float64
	AmountAvailable float64
	AmountFreeze    float64
	AmountFree      float64
	AmountCredit    string
}

func parseBalanceAmount(obj jsonutils.JSONObject, key string) float64 {
	if obj == nil {
		return 0
	}
	if val, err := obj.Float(key); err == nil {
		return val
	}
	if str, err := obj.GetString(key); err == nil && len(str) > 0 {
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f
		}
	}
	return 0
}

func parseBalanceFromResp(resp jsonutils.JSONObject) *SBalance {
	ret := &SBalance{}
	if resp == nil {
		return ret
	}
	info, _ := resp.Get("AccountInfo")
	for _, obj := range []jsonutils.JSONObject{info, resp} {
		if obj == nil {
			continue
		}
		if ret.AmountAvailable == 0 {
			ret.AmountAvailable = parseBalanceAmount(obj, "AmountAvailable")
		}
		if ret.Amount == 0 {
			ret.Amount = parseBalanceAmount(obj, "Amount")
		}
		if ret.AmountFreeze == 0 {
			ret.AmountFreeze = parseBalanceAmount(obj, "AmountFreeze")
		}
		if ret.AmountFree == 0 {
			ret.AmountFree = parseBalanceAmount(obj, "AmountFree")
		}
		if len(ret.AmountCredit) == 0 {
			ret.AmountCredit, _ = obj.GetString("AmountCredit")
		}
	}
	return ret
}

func (balance *SBalance) GetAvailableAmount() float64 {
	if balance.AmountAvailable > 0 {
		return balance.AmountAvailable
	}
	return balance.Amount
}

// QueryBalance 查询账户余额。
func (self *SRockbaseClient) QueryBalance() (*SBalance, error) {
	params := NewRockbaseParams()
	params = self.commonParams(params)
	params.SetAction("GetBalance")
	resp, err := jsonRequest(self, params)
	if err != nil {
		return nil, err
	}
	return parseBalanceFromResp(resp), nil
}

// GetBalance 查询账户余额并转换为 cloudprovider 通用结构。
func (self *SRockbaseClient) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	balance, err := self.QueryBalance()
	if err != nil {
		return nil, errors.Wrap(err, "QueryBalance")
	}
	ret := &cloudprovider.SBalanceInfo{
		Currency: "CNY",
		Amount:   balance.GetAvailableAmount(),
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	if ret.Amount < 0 {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_ARREARS
	}
	if ret.Amount == 0 {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_INSUFFICIENT
	}
	return ret, nil
}
