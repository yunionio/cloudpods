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
	"yunion.io/x/onecloud/pkg/yunionconf/models"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	db.AddScopeResourceCountHandler("", app)

	for _, manager := range []db.IModelManager{
		db.UserCacheManager,
		db.TenantCacheManager,
		db.SharedResourceManager,
		db.OpsLog,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.Metadata,
		models.ParameterManager,
		models.ScopedPolicyBindingManager,
		models.ScopedPolicyManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
		if manager == models.ParameterManager {
			dispatcher.AddModelDispatcher("/users/<user_id>", app, handler)
			dispatcher.AddModelDispatcher("/services/<service_id>", app, handler)
		}
	}
}
