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

type SRouterDeployment struct {
	db.SStandaloneResourceBase

	RouterId          string
	AnsiblePlaybookId string

	RouterRevision int
}

type SRouterDeploymentManager struct {
	db.SStandaloneResourceBaseManager
}

var RouterDeploymentManager *SRouterDeploymentManager

func init() {
	RouterDeploymentManager = &SRouterDeploymentManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SRouterDeployment{},
			"router_deployments_tbl",
			"router_deployment",
			"router_deployments",
		),
	}
	RouterDeploymentManager.SetVirtualObject(RouterDeploymentManager)
}

func (man *SRouterDeploymentManager) requestDeployment(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter) error {
	// XXX
	// make inventory
	// make playbook
	// queue a deploy ansible task
	return nil
}
