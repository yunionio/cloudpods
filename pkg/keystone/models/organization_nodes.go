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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SOrganizationNodeManager struct {
	db.SStandaloneResourceBaseManager
}

var OrganizationNodeManager *SOrganizationNodeManager

func init() {
	OrganizationNodeManager = &SOrganizationNodeManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SOrganizationNode{},
			"organization_nodes_tbl",
			"organization_node",
			"organization_nodes",
		),
	}
	OrganizationNodeManager.SetVirtualObject(OrganizationNodeManager)
}

type SOrganizationNode struct {
	db.SStandaloneResourceBase

	ParentId string `width:"128" charset:"ascii" list:"user" create:"admin_required"`

	Parent string `ignore:"true"`

	RootId string `width:"128" charset:"ascii" list:"user" create:"admin_required"`

	Root string `ignore:"true"`

	Level uint8 `list:"user" create:"admin_required"`

	Labels []string `width:"256" charset:"utf8" list:"user" create:"admin_optional"`
}
