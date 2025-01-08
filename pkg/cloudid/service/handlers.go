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

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/proxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	taskman.InitArchivedTaskManager()
	taskman.AddTaskHandler("v1", app)

	db.AddScopeResourceCountHandler("", app)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		taskman.ArchivedTaskManager,

		db.UserCacheManager,
		db.TenantCacheManager,
		db.SharedResourceManager,
		db.Metadata,
		models.CloudaccountManager,
		models.CloudproviderManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		proxy.ProxySettingManager,
		models.ClouduserManager,
		models.CloudgroupManager,
		models.CloudpolicyManager,
		models.SAMLProviderManager,
		models.CloudroleManager,
		models.SamluserManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
	}

	for _, manager := range []db.IJointModelManager{
		models.ClouduserPolicyManager,
		models.CloudgroupPolicyManager,
		models.CloudgroupUserManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewJointModelHandler(manager)
		dispatcher.AddJointModelDispatcher("", app, handler)
	}

}
