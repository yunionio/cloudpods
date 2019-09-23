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
	"yunion.io/x/onecloud/pkg/ansibleserver/models"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	db.RegisterModelManager(db.OpsLog)
	db.RegisterModelManager(db.UserCacheManager)
	db.RegisterModelManager(db.TenantCacheManager)
	for _, manager := range []db.IModelManager{
		models.AnsiblePlaybookManager,
		models.AnsiblePlaybookV2Manager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
	}
}
