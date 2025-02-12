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

package multicloud

import (
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SBillingBase struct{}

func (self *SBillingBase) GetBillingType() string {
	return ""
}

func (self *SBillingBase) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SBillingBase) SetAutoRenew(bc billing.SBillingCycle) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "SetAutoRenew")
}

func (self *SBillingBase) IsAutoRenew() bool {
	return false
}

func (self *SBillingBase) Renew(bc billing.SBillingCycle) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Renew")
}

func (self *SBillingBase) ChangeBillingType(billType string) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "ChangeBillingType")
}
