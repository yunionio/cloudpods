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

package webconsole

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/webconsole/command"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

const (
	ApiPathPrefix            = "/webconsole/"
	ConnectPathPrefix        = "/connect/"
	WebsockifyPathPrefix     = "/websockify/"
	WebsocketProxyPathPrefix = "/wsproxy/"
)

func InitHandlers(app *appsrv.Application) {
	app.AddHandler("POST", ApiPathPrefix+"k8s/<podName>/shell", auth.Authenticate(handleK8sShell))
	app.AddHandler("POST", ApiPathPrefix+"k8s/<podName>/log", auth.Authenticate(handleK8sLog))
	app.AddHandler("POST", ApiPathPrefix+"baremetal/<id>", auth.Authenticate(handleBaremetalShell))
	app.AddHandler("POST", ApiPathPrefix+"ssh/<ip>", auth.Authenticate(handleSshShell))
	app.AddHandler("POST", ApiPathPrefix+"server/<id>", auth.Authenticate(handleServerRemoteConsole))
}

func fetchK8sEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*command.K8sEnv, error) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	cluster, _ := body.GetString("cluster")
	if cluster == "" {
		cluster = "default"
	}
	namespace, _ := body.GetString("namespace")
	if namespace == "" {
		namespace = "default"
	}
	podName := params["<podName>"]
	container, _ := body.GetString("container")
	adminSession := auth.GetAdminSession(ctx, o.Options.Region, "")

	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(namespace), "namespace")
	query.Add(jsonutils.NewString(cluster), "cluster")
	obj, err := k8s.Pods.Get(adminSession, podName, query)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, httperrors.NewNotFoundError("Not found pod %q", podName)
	}

	data := jsonutils.NewDict()
	ret, err := k8s.KubeClusters.GetSpecific(adminSession, cluster, "kubeconfig", data)
	if err != nil {
		return nil, err
	}
	conf, err := ret.GetString("kubeconfig")
	if err != nil {
		return nil, httperrors.NewNotFoundError("Not found cluster %q kubeconfig", cluster)
	}
	f, err := ioutil.TempFile("", "kubeconfig-")
	if err != nil {
		return nil, fmt.Errorf("Save kubeconfig error: %v", err)
	}
	defer f.Close()
	f.WriteString(conf)

	return &command.K8sEnv{
		Cluster:    cluster,
		Namespace:  namespace,
		Pod:        podName,
		Container:  container,
		Kubeconfig: f.Name(),
		Data:       body,
	}, nil
}

type CloudEnv struct {
	ClientSessin *mcclient.ClientSession
	Params       map[string]string
	Query        jsonutils.JSONObject
	Body         jsonutils.JSONObject
	Ctx          context.Context
}

func fetchCloudEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*CloudEnv, error) {
	params, query, body := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		return nil, httperrors.NewUnauthorizedError("No token founded")
	}
	s := auth.Client().NewSession(ctx, o.Options.Region, "", "internal", userCred, "v2")
	return &CloudEnv{
		ClientSessin: s,
		Params:       params,
		Query:        query,
		Body:         body,
		Ctx:          ctx,
	}, nil
}

func handleK8sCommand(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	cmdFactory func(*command.K8sEnv) command.ICommand,
) {
	env, err := fetchK8sEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	cmd := cmdFactory(env)
	handleCommandSession(cmd, w)
}

func handleK8sShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleK8sCommand(ctx, w, r, command.NewPodBashCommand)
}

func handleK8sLog(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleK8sCommand(ctx, w, r, command.NewPodLogCommand)
}

func handleSshShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	cmd, err := command.NewSSHtoolSolCommand(ctx, userCred, env.Params["<ip>"], env.Body)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	handleCommandSession(cmd, w)
}

func handleBaremetalShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	hostId := env.Params["<id>"]
	ret, err := modules.Hosts.GetSpecific(env.ClientSessin, hostId, "ipmi", nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	info := command.IpmiInfo{}
	err = ret.Unmarshal(&info)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	cmd, err := command.NewIpmitoolSolCommand(&info)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	handleCommandSession(cmd, w)
}

func handleServerRemoteConsole(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	srvId := env.Params["<id>"]
	info, err := session.NewRemoteConsoleInfoByCloud(env.ClientSessin, srvId)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	switch info.Protocol {
	case session.ALIYUN, session.QCLOUD, session.OPENSTACK, session.VMRC, session.ZSTACK:
		responsePublicCloudConsole(info, w)
	case session.VNC, session.SPICE, session.WMKS:
		handleDataSession(info, w, url.Values{"password": {info.GetPassword()}})
	default:
		httperrors.NotAcceptableError(w, "Unspported remote console protocol: %s", info.Protocol)
	}
}

func responsePublicCloudConsole(info *session.RemoteConsoleInfo, w http.ResponseWriter) {
	data := jsonutils.NewDict()
	params, err := info.GetConnectParams()
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	data.Add(jsonutils.NewString(params), "connect_params")
	sendJSON(w, data)
}

func handleDataSession(sData session.ISessionData, w http.ResponseWriter, connParams url.Values) {
	s, err := session.Manager.Save(sData)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	data := jsonutils.NewDict()
	params, err := s.GetConnectParams(connParams)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	data.Add(jsonutils.NewString(params), "connect_params")
	data.Add(jsonutils.NewString(s.Id), "session")
	sendJSON(w, data)
}

func handleCommandSession(cmd command.ICommand, w http.ResponseWriter) {
	handleDataSession(session.WrapCommandSession(cmd), w, nil)
}

func sendJSON(w http.ResponseWriter, body jsonutils.JSONObject) {
	ret := jsonutils.NewDict()
	if body != nil {
		ret.Add(body, "webconsole")
	}
	appsrv.SendJSON(w, ret)
}
