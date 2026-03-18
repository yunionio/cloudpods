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

package aliyun

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SAliyunClient) businessRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "business.aliyuncs.com", ALIYUN_BSS_API_VERSION, apiName, params, self.debug)
}

type SAccountBalance struct {
	AvailableAmount     float64
	AvailableCashAmount float64
	CreditAmount        float64
	MybankCreditAmount  float64
	Currency            string
}

type SCashCoupon struct {
	ApplicableProducts  string
	ApplicableScenarios string
	Balance             float64
	CashCouponId        string
	CashCouponNo        string
	EffectiveTime       time.Time
	ExpiryTime          time.Time
	GrantedTime         time.Time
	NominalValue        float64
	Status              string
}

type SPrepaidCard struct {
	PrepaidCardId       string
	PrepaidCardNo       string
	GrantedTime         time.Time
	EffectiveTime       time.Time
	ExpiryTime          time.Time
	NominalValue        float64
	Balance             float64
	ApplicableProducts  string
	ApplicableScenarios string
}

func (self *SAliyunClient) QueryAccountBalance() (*SAccountBalance, error) {
	body, err := self.businessRequest("QueryAccountBalance", nil)
	if err != nil {
		// {"RequestId":"5258BDEF-8975-4EB0-9E0C-08D5E54E7981","HostId":"business.aliyuncs.com","Code":"NotAuthorized","Message":"This API is not authorized for caller."}
		if isError(err, "NotApplicable") || isError(err, "NotAuthorized") {
			return nil, cloudprovider.ErrNoBalancePermission
		}
		return nil, errors.Wrapf(err, "QueryAccountBalance")
	}
	balance := SAccountBalance{}
	err = body.Unmarshal(&balance, "Data")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal AccountBalance")
	}
	return &balance, nil
}

func (self *SAliyunClient) QueryCashCoupons() ([]SCashCoupon, error) {
	params := make(map[string]string)
	params["EffectiveOrNot"] = "True"
	body, err := self.businessRequest("QueryCashCoupons", params)
	if err != nil {
		return nil, errors.Wrapf(err, "QueryCashCoupons")
	}
	coupons := make([]SCashCoupon, 0)
	err = body.Unmarshal(&coupons, "Data", "CashCoupon")
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal")
	}
	return coupons, nil
}

func (self *SAliyunClient) QueryPrepaidCards() ([]SPrepaidCard, error) {
	params := make(map[string]string)
	params["EffectiveOrNot"] = "True"
	body, err := self.businessRequest("QueryPrepaidCards", params)
	if err != nil {
		return nil, errors.Wrapf(err, "QueryPrepaidCards")
	}
	cards := make([]SPrepaidCard, 0)
	err = body.Unmarshal(&cards, "Data", "PrepaidCard")
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal")
	}
	return cards, nil
}

func (self *SAliyunClient) SubscribeBillToOSS(bucket string) error {
	params := make(map[string]string)
	params["SubscribeBucket"] = bucket
	params["SubscribeType.0"] = "BillingItemDetailForBillingPeriod"
	params["SubscribeType.1"] = "InstanceDetailForBillingPeriod"
	body, err := self.businessRequest("SubscribeBillToOSS", params)
	if err != nil {
		return errors.Wrap(err, "SubscribeBillToOSS")
	}
	log.Debugf("%s", body)
	return nil
}

type SBillOverview struct {
	BillingCycle string     `json:"BillingCycle"`
	AccountID    string     `json:"AccountID"`
	AccountName  string     `json:"AccountName"`
	Items        SBillItems `json:"Items"`
	TotalAmount  float64    `json:"TotalAmount"`
}

type SBillItems struct {
	Item []SBillItem `json:"Item"`
}

type SBillItem struct {
	DeductedByCoupons     float64 `json:"DeductedByCoupons"`
	RoundDownDiscount     float64 `json:"RoundDownDiscount"`
	ProductName           string  `json:"ProductName"`
	ProductDetail         string  `json:"ProductDetail"`
	ProductCode           string  `json:"ProductCode"`
	BillAccountID         string  `json:"BillAccountID"`
	ProductType           string  `json:"ProductType"`
	DeductedByCashCoupons float64 `json:"DeductedByCashCoupons"`
	OutstandingAmount     float64 `json:"OutstandingAmount"`
	BizType               string  `json:"BizType"`
	PaymentAmount         float64 `json:"PaymentAmount"`
	PipCode               string  `json:"PipCode"`
	DeductedByPrepaidCard float64 `json:"DeductedByPrepaidCard"`
	InvoiceDiscount       float64 `json:"InvoiceDiscount"`
	Item                  string  `json:"Item"`
	SubscriptionType      string  `json:"SubscriptionType"`
	PretaxGrossAmount     float64 `json:"PretaxGrossAmount"`
	PretaxAmount          float64 `json:"PretaxAmount"`
	OwnerID               string  `json:"OwnerID"`
	Currency              string  `json:"Currency"`
	CommodityCode         string  `json:"CommodityCode"`
	BillAccountName       string  `json:"BillAccountName"`
	AdjustAmount          float64 `json:"AdjustAmount"`
	CashAmount            float64 `json:"CashAmount"`
}

func (self *SAliyunClient) QueryBillOverview(billCycle, ownerId string) (*SBillOverview, error) {
	params := make(map[string]string)
	params["BillingCycle"] = billCycle
	if len(ownerId) > 0 {
		params["BillOwnerId"] = ownerId
	}
	body, err := self.businessRequest("QueryBillOverview", params)
	if err != nil {
		return nil, errors.Wrap(err, "QueryBillOverview")
	}
	overview := SBillOverview{}
	err = body.Unmarshal(&overview, "Data")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal")
	}
	totalAmount := 0.0
	for _, item := range overview.Items.Item {
		totalAmount += item.PretaxAmount
	}
	overview.TotalAmount = totalAmount
	return &overview, nil
}
