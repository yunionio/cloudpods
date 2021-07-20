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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/oldmodels"
)

const (
	API_VERSION = "v2"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	db.RegistUserCredCacheUpdater()

	db.AddScopeResourceCountHandler(API_VERSION, app)

	// Data migration
	db.RegisterModelManager(oldmodels.NotificationManager)
	db.RegisterModelManager(oldmodels.ContactManager)
	db.RegisterModelManager(oldmodels.ConfigManager)
	db.RegisterModelManager(oldmodels.TemplateManager)
	db.RegisterModelManager(oldmodels.UserCacheManager)

	taskman.AddTaskHandler(API_VERSION, app)
	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,

		db.UserCacheManager,
		db.TenantCacheManager,
		db.RoleCacheManager,
		models.SubContactManager,
		db.SharedResourceManager,
		models.VerificationManager,
		models.EventManager,
	} {
		db.RegisterModelManager(manager)
	}
	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,

		models.ReceiverManager,
		models.NotificationManager,
		models.ConfigManager,
		models.TemplateManager,
		models.TopicManager,
		models.RobotManager,
		models.SubscriberManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher(API_VERSION, app, handler)
	}
	for _, manager := range []db.IJointModelManager{
		models.ReceiverNotificationManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewJointModelHandler(manager)
		dispatcher.AddJointModelDispatcher(API_VERSION, app, handler)
	}
}
