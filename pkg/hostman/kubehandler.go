package hostman

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type sKubeConf struct {
	DockerdConf map[string]interface{}
	AgentConfig map[string]interface{}
}

var keyWords = []string{"kubeagent"}

func addKubeAgentHandler(prefix string, app *appsrv.Application) {
	for _, keyword := range keyWords {
		app.AddHandler("POST", fmt.Sprintf("%s/%s/<action>", prefix, keyword),
			auth.Authenticate(dispatcher))
	}
}

func dispatcher(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var (
		params, _, body = appsrv.FetchEnv(ctx, w, r)
		action          = params["<action>"]
	)
	switch action {
	case "start":
		dockerdConf, _ := body.Get("dockerdConfig")
		agentConfig, _ := body.Get("agentConfig")
		err := prepareAgentStart(dockerdConf, agentConfig)
		if err != nil {
			hostutils.Response(ctx, w, err)
			return
		}
		var dm = map[string]interface{}{}
		if err := dockerdConf.Unmarshal(&dm); err != nil {
			hostutils.Response(ctx, w, err)
			return
		}

		var am = map[string]interface{}{}
		if err := agentConfig.Unmarshal(&am); err != nil {
			hostutils.Response(ctx, w, err)
			return
		}

		hostutils.DelayTask(ctx, startAgent, &sKubeConf{dm, am})
	case "restart":
		hostutils.DelayTask(ctx, restartAgent, nil)
	case "stop":
		hostutils.DelayTask(ctx, stopAgent, nil)
	default:
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Not found"))
		return
	}
	hostutils.ResponseOk(ctx, w)
}

func prepareAgentStart(dockerdConf, agentConfig jsonutils.JSONObject) error {
	if !agentConfig.Contains("serverUrl") {
		return httperrors.NewBadRequestError("Kube server url empty")
	}
	if !agentConfig.Contains("nodeId") {
		return httperrors.NewBadRequestError("NodeId empty")
	}
	if !agentConfig.Contains("token") {
		return httperrors.NewBadRequestError("Register token empty")
	}
	return nil
}

func startAgent(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sp, ok := params.(*sKubeConf)
	if !ok {
		return nil, hostutils.ParamsError
	}
	lxcfs := system_service.GetService("lxcfs")
	if !lxcfs.IsInstalled() {
		return nil, fmt.Errorf("Service lxcfs not installed")
	} else if !lxcfs.IsActive() {
		if err := lxcfs.Start(false); err != nil {
			return nil, err
		}
	}
	if err := lxcfs.Enable(); err != nil {
		return nil, err
	}
	if err := serviceReloadStart("docker", sp.DockerdConf); err != nil {
		return nil, err
	}
	if err := serviceReloadStart("kube_agent", sp.AgentConfig); err != nil {
		return nil, err
	}
	return nil, nil
}

func serviceReloadStart(srv string, conf map[string]interface{}) error {
	srvinst := system_service.GetService(srv)
	if srvinst == nil {
		return fmt.Errorf("srv %s not found", srv)
	}
	if !srvinst.IsInstalled() {
		return fmt.Errorf("Service %s nout found", srv)
	}
	if err := srvinst.Reload(conf); err != nil {
		return err
	}
	if !srvinst.IsActive() {
		if err := srvinst.Start(false); err != nil {
			return err
		}
	}
	return srvinst.Enable()
}

func restartAgent(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	srvinst := system_service.GetService("kube_agent")
	if !srvinst.IsInstalled() {
		return nil, fmt.Errorf("Service kube_agent not installed")
	}
	return nil, srvinst.Start(true)
}

func stopAgent(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	srvinst := system_service.GetService("kube_agent")
	if !srvinst.IsInstalled() {
		return nil, fmt.Errorf("Service kube_agent not installed")
	}
	return nil, srvinst.Stop(true)
}
