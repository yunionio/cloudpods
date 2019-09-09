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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/usages"
)

const (
	API_VERSION = "v1"
)

func initHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	// add version handler with API_VERSION prefix
	app.AddDefaultHandler("GET", API_VERSION+"/version", appsrv.VersionHandler, "version")

	db.RegistUserCredCacheUpdater()

	db.AddProjectResourceCountHandler(API_VERSION, app)

	quotas.AddQuotaHandler(&models.QuotaManager.SQuotaBaseManager, API_VERSION, app)
	usages.AddUsageHandler(API_VERSION, app)
	taskman.AddTaskHandler(API_VERSION, app)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		db.UserCacheManager,
		db.TenantCacheManager,
		db.SharedResourceManager,
		models.ImageTagManager,
		models.ImageMemberManager,
		models.ImagePropertyManager,
		models.ImageSubformatManager,

		models.GuestImageJointManager,

		models.QuotaManager,
		models.QuotaUsageManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,
		models.ImageManager,

		models.GuestImageManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher(API_VERSION, app, handler)
	}
}
