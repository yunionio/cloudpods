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
	api "yunion.io/x/cloudmux/pkg/apis/compute"
)

type SSubscription struct {
	SubscriptionId string
	State          string
	DisplayName    string
}

func (self *SSubscription) GetHealthStatus() string {
	if self.State == "Enabled" {
		return api.CLOUD_PROVIDER_HEALTH_NORMAL
	}
	return api.CLOUD_PROVIDER_HEALTH_SUSPENDED
}

func (self *SAzureClient) ListSubscriptions() ([]SSubscription, error) {
	resp, err := self.list_v2("subscriptions", "2014-02-26", nil)
	if err != nil {
		return nil, err
	}
	result := []SSubscription{}
	err = resp.Unmarshal(&result, "value")
	if err != nil {
		return nil, err
	}
	return result, nil
}
