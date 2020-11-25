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

type SInterVpcNetworkVpcManager struct {
	db.SJointResourceBaseManager
	SInterVpcNetworkResourceBaseManager
}

var InterVpcNetworkVpcManager *SInterVpcNetworkVpcManager

func init() {
	db.InitManager(func() {
		InterVpcNetworkVpcManager = &SInterVpcNetworkVpcManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SInterVpcNetworkVpc{},
				"inter_vpc_network_vpc_tbl",
				"inter_vpc_network_vpc",
				"inter_vpc_network_vpcs",
				InterVpcNetworkManager,
				VpcManager,
			),
		}
		InterVpcNetworkManager.SetVirtualObject(InterVpcNetworkManager)
	})
}

type SInterVpcNetworkVpc struct {
	db.SJointResourceBase
	SInterVpcNetworkResourceBase

	VpcId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SInterVpcNetworkVpcManager) GetMasterFieldName() string {
	return "inter_vpc_network_id"
}

func (manager *SInterVpcNetworkVpcManager) GetSlaveFieldName() string {
	return "vpc_id"
}

func (self *SInterVpcNetworkVpc) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}
