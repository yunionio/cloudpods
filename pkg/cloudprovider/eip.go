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

type TElasticipChargeType string

var (
	ElasticipChargeTypeByTraffic   = TElasticipChargeType("traffic")
	ElasticipChargeTypeByBandwidth = TElasticipChargeType("bandwidth")
)

type SEip struct {
	Name              string
	BandwidthMbps     int
	ChargeType        string
	BGPType           string
	NetworkExternalId string
	IP                string
	ProjectId         string
}

type AssociateConfig struct {
	InstanceId    string
	AssociateType string
	Bandwidth     int
}
