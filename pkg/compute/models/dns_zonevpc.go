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

type SDnsZoneVpcManager struct {
	db.SJointResourceBaseManager
	SDnsZoneResourceBaseManager
}

var DnsZoneVpcManager *SDnsZoneVpcManager

func init() {
	db.InitManager(func() {
		DnsZoneVpcManager = &SDnsZoneVpcManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SDnsZoneVpc{},
				"dns_zonevpcs_tbl",
				"dns_zonevpc",
				"dns_zonevpcs",
				DnsZoneManager,
				VpcManager,
			),
		}
		DnsZoneVpcManager.SetVirtualObject(DnsZoneVpcManager)
	})
}

type SDnsZoneVpc struct {
	db.SJointResourceBase
	SDnsZoneResourceBase

	VpcId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SDnsZoneVpcManager) GetMasterFieldName() string {
	return "dns_zone_id"
}

func (manager *SDnsZoneVpcManager) GetSlaveFieldName() string {
	return "vpc_id"
}

func (self *SDnsZoneVpc) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}
