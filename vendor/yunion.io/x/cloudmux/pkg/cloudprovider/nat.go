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

package cloudprovider

import "yunion.io/x/pkg/util/billing"

// These two structures are designed for modifying snat table and dnat table.
// There is a so strange point that they have both field of ExternalIP and ExternalIPID.
// The reason is that you must pass ExternalIPID to modify in Huawei Cloud for now.
// So please construct a valid parameter ExternalIPID instead of ExternalIP in Huawei Cloud.
// A more general approach is to pass both valid parameters.

type SNatSRule struct {
	SourceCIDR string
	NetworkID  string

	ExternalIP   string
	ExternalIPID string
}

type SNatDRule struct {
	Protocol string

	InternalIP   string
	InternalPort int

	ExternalIP   string
	ExternalIPID string
	ExternalPort int
}

type NatGatewayCreateOptions struct {
	Name      string
	VpcId     string
	NetworkId string
	Desc      string
	NatSpec   string

	BillingCycle *billing.SBillingCycle
}
