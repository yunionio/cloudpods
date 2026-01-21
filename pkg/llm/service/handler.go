package service

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func handleOllamaRegistryYAML(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	yamlContent := models.GetInstantModelManager().GetOllamaRegistryYAML()
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	appsrv.Send(w, yamlContent)
}

func AddAvailableNetworkHandler(prefix string, app *appsrv.Application) {
	app.AddHandler2("GET", fmt.Sprintf("%s/available-network", prefix), auth.Authenticate(handleLLMAvailableNetwork), nil, "get_llm_available_network", nil)
}

func handleLLMAvailableNetwork(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "Parse query string %q: %v", r.URL.RawQuery, err)
		return
	}
	ret, err := models.GetLLMManager().GetAvailableNetwork(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	wrapped := jsonutils.NewDict()
	if ret != nil {
		wrapped.Add(ret, "llm")
	}
	appsrv.SendJSON(w, wrapped)
}

func InitHandlers(app *appsrv.Application, isSlave bool) {
	db.InitAllManagers()
	db.RegistUserCredCacheUpdater()

	taskman.AddTaskHandler("", app, isSlave)

	app.AddHandler("GET", "/ollama-registry.yaml", handleOllamaRegistryYAML)

	AddAvailableNetworkHandler(models.GetLLMManager().KeywordPlural(), app)

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
