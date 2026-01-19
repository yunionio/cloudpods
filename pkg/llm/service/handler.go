package service

import (
	"context"
	"net/http"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func handleOllamaRegistryYAML(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	yamlContent := models.GetInstantModelManager().GetOllamaRegistryYAML()
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	appsrv.Send(w, yamlContent)
}

func InitHandlers(app *appsrv.Application, isSlave bool) {
	db.InitAllManagers()
	db.RegistUserCredCacheUpdater()

	taskman.AddTaskHandler("", app, isSlave)

	app.AddHandler("GET", "/ollama-registry.yaml", handleOllamaRegistryYAML)

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

		models.GetLLMImageManager(),
		models.GetLLMSkuManager(),
		models.GetDifySkuManager(),
		models.GetVolumeManager(),
		models.GetAccessInfoManager(),
		models.GetLLMContainerManager(),
		models.GetLLMManager(),
		models.GetDifyManager(),
		models.GetInstantModelManager(),
		models.GetLLMInstantModelManager(),
		models.GetMCPAgentManager(),
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler, isSlave)
	}
}
