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

package oracle

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SOrganizationSubscriptionCurrency struct {
	IsoCode string
	Name    string
}

type SOrganizationSubscription struct {
	Id          string
	ServiceName string
	Type        string
	Status      string
	TimeStart   time.Time
	TimeEnd     time.Time
	Currency    SOrganizationSubscriptionCurrency
	TotalValue  string
}

type SSubscribedService struct {
	Id              string
	SubscriptionId  string
	Status          string
	IsAllowance     bool
	UsedAmount      string
	AvailableAmount string
	TimeStart       time.Time
	TimeEnd         time.Time
}

type SCommitment struct {
	Id                    string
	SubscribedServiceId   string
	TimeStart             time.Time
	TimeEnd               time.Time
	Quantity              string
	UsedAmount            string
	AvailableAmount       string
	FundedAllocationValue string
}

func parseAmount(val string) float64 {
	if len(val) == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}
	return f
}

func (s SSubscribedService) isActive() bool {
	return strings.EqualFold(s.Status, "ACTIVE")
}

// listOneSubscription lists OneSubscription resources. compartmentId must be the tenancy (root) OCID.
func (self *SOracleClient) listOneSubscription(resource string, query url.Values) (jsonutils.JSONObject, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("compartmentId", self.tenancyOCID)
	region, _ := self.Region()
	ret := jsonutils.NewArray()
	for {
		resp, token, err := self.request(httputils.GET, SERVICE_ONESUBSCRIPTION, region, resource, query, nil)
		if err != nil {
			return nil, err
		}
		items, _ := resp.GetArray()
		ret.Add(items...)
		if len(token) == 0 {
			break
		}
		query.Set("page", token)
	}
	return ret, nil
}

func (self *SOracleClient) GetOrganizationSubscriptions() ([]SOrganizationSubscription, error) {
	resp, err := self.listOneSubscription("organizationSubscription", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "list organizationSubscription")
	}
	ret := []SOrganizationSubscription{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SOracleClient) GetSubscribedServices(subscriptionId string) ([]SSubscribedService, error) {
	query := url.Values{}
	query.Set("subscriptionId", subscriptionId)
	resp, err := self.listOneSubscription("subscribedServices", query)
	if err != nil {
		return nil, errors.Wrapf(err, "list subscribedServices")
	}
	ret := []SSubscribedService{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SOracleClient) GetCommitments(subscribedServiceId string) ([]SCommitment, error) {
	query := url.Values{}
	query.Set("subscribedServiceId", subscribedServiceId)
	resp, err := self.listOneSubscription("commitments", query)
	if err != nil {
		return nil, errors.Wrapf(err, "list commitments")
	}
	ret := []SCommitment{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

// GetCommitmentsBalance lists commitments under ACTIVE subscribed services.
// Commitment timeEnd is a billing-period boundary and must not be used to drop remaining credits.
func (self *SOracleClient) GetCommitmentsBalance() ([]SCommitment, error) {
	subs, err := self.GetOrganizationSubscriptions()
	if err != nil {
		return nil, err
	}
	ret := []SCommitment{}
	for _, sub := range subs {
		services, err := self.GetSubscribedServices(sub.Id)
		if err != nil {
			return nil, errors.Wrapf(err, "GetSubscribedServices %s", sub.Id)
		}
		for _, svc := range services {
			if !svc.isActive() {
				continue
			}
			commitments, err := self.GetCommitments(svc.Id)
			if err != nil {
				return nil, errors.Wrapf(err, "GetCommitments %s", svc.Id)
			}
			ret = append(ret, commitments...)
		}
	}
	return ret, nil
}

// GetBalance aggregates subscribed service availableAmount under ACTIVE lines.
// Prefer service-level availableAmount over commitment: commitment figures can lag
// usage and commitment timeEnd is only a billing-period boundary.
func (self *SOracleClient) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{
		Currency: "USD",
		Status:   api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	subs, err := self.GetOrganizationSubscriptions()
	if err != nil {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
		return ret, err
	}
	var amount float64
	for _, sub := range subs {
		if len(sub.Currency.IsoCode) > 0 {
			ret.Currency = sub.Currency.IsoCode
		}
		services, err := self.GetSubscribedServices(sub.Id)
		if err != nil {
			ret.Status = api.CLOUD_PROVIDER_HEALTH_UNKNOWN
			return ret, errors.Wrapf(err, "GetSubscribedServices %s", sub.Id)
		}
		for _, svc := range services {
			if !svc.isActive() {
				continue
			}
			amount += parseAmount(svc.AvailableAmount)
		}
	}
	ret.Amount = amount
	if ret.Amount < 0 {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_ARREARS
	} else if ret.Amount == 0 {
		ret.Status = api.CLOUD_PROVIDER_HEALTH_INSUFFICIENT
	}
	return ret, nil
}
