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

package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDnsRecordSetTrafficPolicyManager struct {
	db.SJointResourceBaseManager
	SDnsZoneResourceBaseManager
}

var DnsRecordSetTrafficPolicyManager *SDnsRecordSetTrafficPolicyManager

func init() {
	db.InitManager(func() {
		DnsRecordSetTrafficPolicyManager = &SDnsRecordSetTrafficPolicyManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SDnsRecordSetTrafficPolicy{},
				"dns_recordset_trafficpolicy_tbl",
				"dns_recordset_trafficpolicy",
				"dns_recordset_trafficpolicies",
				DnsRecordSetManager,
				DnsTrafficPolicyManager,
			),
		}
		DnsRecordSetTrafficPolicyManager.SetVirtualObject(DnsRecordSetTrafficPolicyManager)
	})
}

type SDnsRecordSetTrafficPolicy struct {
	db.SJointResourceBase
	DnsRecordsetId string `width:"36" charset:"ascii" nullable:"false" list:"user" json:"dns_recordset_id"`

	DnsTrafficPolicyId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SDnsRecordSetTrafficPolicyManager) GetMasterFieldName() string {
	return "dns_recordset_id"
}

func (manager *SDnsRecordSetTrafficPolicyManager) GetSlaveFieldName() string {
	return "dns_traffic_policy_id"
}

func (self *SDnsRecordSetTrafficPolicy) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}
