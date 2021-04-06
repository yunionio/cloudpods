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

package service

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudproxy/models"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	db.RegisterModelManager(db.OpsLog)
	db.RegisterModelManager(db.Metadata)
	db.RegisterModelManager(db.TenantCacheManager)
	db.RegisterModelManager(db.UserCacheManager)
	for _, manager := range []db.IModelManager{
		models.ProxyEndpointManager,
		models.ProxyMatchManager,
		models.ProxyAgentManager,
		models.ForwardManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
	}
}
