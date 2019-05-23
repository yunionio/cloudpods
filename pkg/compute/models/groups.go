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

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

const (
	REDIS_TYPE = "REDIS"
	RDS_TYPE   = "RDS"
)

type SGroupManager struct {
	db.SVirtualResourceBaseManager
}

var GroupManager *SGroupManager

func init() {
	GroupManager = &SGroupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SGroup{},
			"groups_tbl",
			"group",
			"groups",
		),
	}
	GroupManager.SetVirtualObject(GroupManager)
}

type SGroup struct {
	db.SVirtualResourceBase

	ServiceType string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	ParentId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	ZoneId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	SchedStrategy string `width:"16" charset:"ascii" nullable:"true" default:"" list:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')
}

func (group *SGroup) GetNetworks() ([]SGroupnetwork, error) {
	q := GroupnetworkManager.Query().Equals("group_id", group.Id)
	groupnets := make([]SGroupnetwork, 0)
	err := db.FetchModelObjects(GroupnetworkManager, q, &groupnets)
	if err != nil {
		return nil, err
	}
	return groupnets, nil
}
