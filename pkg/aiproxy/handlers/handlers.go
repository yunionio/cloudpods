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

package handlers

import (
	"time"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/options"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	openaiCompatAPIPrefix     = "/ai/openai/v1"
	openaiLongProcessTimeout  = 2 * time.Hour
	openaiShortProcessTimeout = 5 * time.Minute
)

func InitHandlers(app *appsrv.Application, isSlave bool) {
	db.InitAllManagers()
	db.RegistUserCredCacheUpdater()

	app_common.ExportOptionsHandler(app, &options.Options)

	taskman.AddTaskHandler("", app, isSlave)

	db.AddScopeResourceCountHandler("", app)

	app.AddHandler2("POST", openaiCompatAPIPrefix+"/chat/completions", chatCompletionsHandler, nil, "aiproxy_openai_v1_chat_completions", nil).
		SetProcessTimeout(openaiLongProcessTimeout)
	app.AddHandler2("POST", openaiCompatAPIPrefix+"/completions", completionsHandler, nil, "aiproxy_openai_v1_completions", nil).
		SetProcessTimeout(openaiLongProcessTimeout)
	app.AddHandler2("POST", openaiCompatAPIPrefix+"/embeddings", embeddingsHandler, nil, "aiproxy_openai_v1_embeddings", nil).
		SetProcessTimeout(openaiShortProcessTimeout)
	app.AddHandler2("POST", openaiCompatAPIPrefix+"/images/generations", imagesGenerationsHandler, nil, "aiproxy_openai_v1_images_generations", nil).
		SetProcessTimeout(openaiShortProcessTimeout)
	app.AddHandler2("GET", openaiCompatAPIPrefix+"/models", modelsHandler, nil, "aiproxy_openai_v1_models", nil)
	app.AddHandler2("GET", openaiCompatAPIPrefix+"/models/<model>", modelRetrieveHandler, nil, "aiproxy_openai_v1_models_retrieve", nil)
	app.AddHandler2("GET", "/usage/overview", auth.Authenticate(usageOverviewHandler), nil, "aiproxy_usage_overview", nil)
	app.AddHandler2("GET", "/usage/analysis", auth.Authenticate(usageAnalysisHandler), nil, "aiproxy_usage_analysis", nil)
	app.AddHandler2("GET", "/usage/events", auth.Authenticate(usageEventsHandler), nil, "aiproxy_usage_events", nil)
	app.AddHandler2("GET", "/usage/api-keys/options", auth.Authenticate(usageAPIKeysOptionsHandler), nil, "aiproxy_usage_api_keys_options", nil)
	app.AddHandler2("GET", "/api/v1/usage/overview", auth.Authenticate(usageOverviewHandler), nil, "aiproxy_api_v1_usage_overview", nil)
	app.AddHandler2("GET", "/api/v1/usage/analysis", auth.Authenticate(usageAnalysisHandler), nil, "aiproxy_api_v1_usage_analysis", nil)
	app.AddHandler2("GET", "/api/v1/usage/events", auth.Authenticate(usageEventsHandler), nil, "aiproxy_api_v1_usage_events", nil)
	app.AddHandler2("GET", "/api/v1/usage/api-keys/options", auth.Authenticate(usageAPIKeysOptionsHandler), nil, "aiproxy_api_v1_usage_api_keys_options", nil)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		taskman.ArchivedTaskManager,

		db.SharedResourceManager,
		db.UserCacheManager,
		db.TenantCacheManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,

		models.AiProviderManager,
		models.AiModelManager,
		models.AiKeyManager,
		models.AiVirtualKeyManager,
		models.AiRoutingManager,
		models.AiRoutingModelManager,
		models.AiProxyNodeManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler, isSlave)
	}
}
