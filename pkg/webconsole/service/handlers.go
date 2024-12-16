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
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/regutils"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	webconsole_api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/webconsole/command"
	"yunion.io/x/onecloud/pkg/webconsole/models"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/server"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

const (
	ApiPathPrefix            = "/webconsole/"
	ConnectPathPrefix        = "/connect/"
	WebsockifyPathPrefix     = "/websockify/"
	WebsocketProxyPathPrefix = "/wsproxy/"
)

func initHandlers(app *appsrv.Application) {
	app.AddHandler("POST", ApiPathPrefix+"k8s/<podName>/shell", auth.Authenticate(handleK8sShell))
	app.AddHandler("POST", ApiPathPrefix+"climc/shell", auth.Authenticate(handleClimcShell))
	app.AddHandler("POST", ApiPathPrefix+"k8s/<podName>/log", auth.Authenticate(handleK8sLog))
	app.AddHandler("POST", ApiPathPrefix+"baremetal/<id>", auth.Authenticate(handleBaremetalShell))
	app.AddHandler("POST", ApiPathPrefix+"ssh/<ip>", auth.Authenticate(handleSshShell))
	app.AddHandler("POST", ApiPathPrefix+"server/<id>", auth.Authenticate(handleServerRemoteConsole))
	app.AddHandler("POST", ApiPathPrefix+"adb/<id>/shell", auth.Authenticate(handleAdbShell))
	app.AddHandler("POST", ApiPathPrefix+"server-rdp/<id>", auth.Authenticate(handleServerRemoteRDPConsole))
	app.AddHandler("GET", ApiPathPrefix+"sftp/<session-id>/list", server.HandleSftpList)
	app.AddHandler("GET", ApiPathPrefix+"sftp/<session-id>/download", server.HandleSftpDownload)
	app.AddHandler("POST", ApiPathPrefix+"sftp/<session-id>/upload", server.HandleSftpUpload)

	for _, man := range []db.IModelManager{
		models.GetCommandLogManager(),
	} {
		db.RegisterModelManager(man)
		handler := db.NewModelHandler(man)
		dispatcher.AddModelDispatcher(ApiPathPrefix, app, handler)
	}
}

func fetchK8sEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*command.K8sEnv, error) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	if !gotypes.IsNil(body) && body.Contains("webconsole") {
		body, _ = body.Get("webconsole")
	}

	k8sReq := webconsole_api.SK8sRequest{}
	err := body.Unmarshal(&k8sReq)
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal SK8sRequest")
	}

	if k8sReq.Cluster == "" {
		k8sReq.Cluster = "default"
	}
	if k8sReq.Namespace == "" {
		k8sReq.Namespace = "default"
	}
	podName := params["<podName>"]
	adminSession := auth.GetAdminSession(ctx, o.Options.Region)

	data := jsonutils.NewDict()
	ret, err := k8s.KubeClusters.GetSpecific(adminSession, k8sReq.Cluster, "kubeconfig", data)
	if err != nil {
		return nil, err
	}
	conf, err := ret.GetString("kubeconfig")
	if err != nil {
		return nil, httperrors.NewNotFoundError("Not found cluster %q kubeconfig", k8sReq.Cluster)
	}
	f, err := os.CreateTemp("", "kubeconfig-")
	if err != nil {
		return nil, fmt.Errorf("Save kubeconfig error: %v", err)
	}
	defer f.Close()
	f.WriteString(conf)

	return &command.K8sEnv{
		Session:    adminSession,
		Cluster:    k8sReq.Cluster,
		Namespace:  k8sReq.Namespace,
		Pod:        podName,
		Container:  k8sReq.Container,
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
	if !gotypes.IsNil(body) && body.Contains("webconsole") {
		body, _ = body.Get("webconsole")
	}
	s := auth.Client().NewSession(ctx, o.Options.Region, "", "internal", userCred)
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
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	cmd := cmdFactory(env)
	handleCommandSession(ctx, cmd, w)
}

func handleK8sShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleK8sCommand(ctx, w, r, command.NewPodBashCommand)
}

func handleK8sLog(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleK8sCommand(ctx, w, r, command.NewPodLogCommand)
}

func handleSshShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	sshConnInfo := session.SSshConnectionInfo{}
	if !gotypes.IsNil(env.Body) {
		err = env.Body.Unmarshal(&sshConnInfo)
		if err != nil {
			httperrors.InputParameterError(ctx, w, "unmarshal error: %s", err.Error())
			return
		}
	}
	idStr := env.Params["<ip>"]
	if !regutils.MatchIPAddr(idStr) {
		var tryServer = func() error {
			ip, port, guestDetails, err := session.ResolveServerSSHIPPortById(ctx, env.ClientSessin, idStr, sshConnInfo.IP, sshConnInfo.Port)
			if err != nil {
				return err
			}
			sshConnInfo.IP = ip
			sshConnInfo.Port = port
			sshConnInfo.GuestDetails = guestDetails
			return nil
		}
		var tryHost = func() error {
			ip, port, hostDetails, err := session.ResolveHostSSHIPPortById(ctx, env.ClientSessin, idStr, sshConnInfo.IP, sshConnInfo.Port)
			if err != nil {
				return err
			}
			sshConnInfo.IP = ip
			sshConnInfo.Port = port
			sshConnInfo.HostDetails = hostDetails
			return nil
		}
		switch sshConnInfo.ResourceType {
		case "server":
			err = tryServer()
			if err != nil {
				httperrors.GeneralServerError(ctx, w, err)
				return
			}
		case "host":
			err = tryHost()
			if err != nil {
				httperrors.GeneralServerError(ctx, w, err)
				return
			}
		default:
			for _, try := range []func() error{
				tryServer,
				tryHost,
			} {
				err = try()
				if err == nil {
					s := session.NewSshSession(ctx, env.ClientSessin, sshConnInfo)
					handleSshSession(ctx, s, w)
					return
				}
			}
			httperrors.NewResourceNotFoundError("%s not found", idStr)
			return
		}
	} else {
		// directly ssh IP should be deprecated gradually
		sshConnInfo.IP = idStr
	}
	s := session.NewSshSession(ctx, env.ClientSessin, sshConnInfo)
	handleSshSession(ctx, s, w)
}

func handleSshSession(ctx context.Context, session *session.SSshSession, w http.ResponseWriter) {
	handleDataSession(ctx, session, w, "ws", nil, false)
}

func handleBaremetalShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	hostId := env.Params["<id>"]
	ret, err := modules.Hosts.GetSpecific(env.ClientSessin, hostId, "ipmi", nil)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	info := command.IpmiInfo{}
	err = ret.Unmarshal(&info)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	cmd, err := command.NewIpmitoolSolCommand(&info, env.ClientSessin)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	handleCommandSession(ctx, cmd, w)
}

func handleClimcShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	info := webconsole_api.ClimcSshInfo{}
	err = env.Body.Unmarshal(&info)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	cmd, err := command.NewClimcSshCommand(&info, env.ClientSessin)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	handleCommandSession(ctx, cmd, w)
}

func handleServerRemoteConsole(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	query := env.Body
	srvId := env.Params["<id>"]
	info, err := session.NewRemoteConsoleInfoByCloud(env.ClientSessin, srvId, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	switch info.Protocol {
	case session.ALIYUN, session.QCLOUD, session.OPENSTACK,
		session.VMRC, session.ZSTACK, session.CTYUN,
		session.HUAWEI, session.HCS, session.APSARA,
		session.JDCLOUD, session.CLOUDPODS, session.PROXMOX,
		session.VOLCENGINE, session.BAIDU:
		responsePublicCloudConsole(ctx, info, w)
	case session.VNC:
		handleDataSession(ctx, info, w, "no-vnc", url.Values{"password": {info.GetPassword()}}, true)
	case session.SPICE:
		handleDataSession(ctx, info, w, "spice", url.Values{"password": {info.GetPassword()}}, true)
	case session.WMKS:
		handleDataSession(ctx, info, w, "wmks", url.Values{"password": {info.GetPassword()}}, true)
	default:
		httperrors.NotAcceptableError(ctx, w, "Unspported remote console protocol: %s", info.Protocol)
	}
}

func handleServerRemoteRDPConsole(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	query := env.Body
	srvId := env.Params["<id>"]
	info, err := session.NewRemoteRDPConsoleInfoByCloud(ctx, env.ClientSessin, srvId, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	handleDataSession(ctx, info, w, "rdp", url.Values{"password": {info.GetPassword()}}, true)
}

func responsePublicCloudConsole(ctx context.Context, info *session.RemoteConsoleInfo, w http.ResponseWriter) {
	params, err := info.GetConnectParams()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	resp := webconsole_api.ServerRemoteConsoleResponse{
		ConnectParams: params,
	}
	sendJSON(w, resp.JSON(resp))
}

func handleDataSession(ctx context.Context, sData session.ISessionData, w http.ResponseWriter, base string, connParams url.Values, b64Encode bool) {
	s, err := session.Manager.Save(sData)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	dispInfo, err := sData.GetDisplayInfo(ctx)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	params, err := s.GetConnectParams(connParams, dispInfo)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	var accessUrl string
	{
		dataVal := url.Values{}
		dataVal.Add("data", base64.StdEncoding.EncodeToString([]byte(params)))
		accessUrl = httputils.JoinPath(o.Options.ApiServer, fmt.Sprintf("web-console/%s?%s", base, dataVal.Encode()))
	}

	if b64Encode {
		params = base64.StdEncoding.EncodeToString([]byte(params))
	}

	resp := webconsole_api.ServerRemoteConsoleResponse{
		AccessUrl:     accessUrl,
		ConnectParams: params,
		Session:       s.Id,
	}
	sendJSON(w, resp.JSON(resp))
}

func handleCommandSession(ctx context.Context, cmd command.ICommand, w http.ResponseWriter) {
	handleDataSession(ctx, session.WrapCommandSession(cmd), w, "tty", nil, false)
}

func sendJSON(w http.ResponseWriter, body jsonutils.JSONObject) {
	ret := jsonutils.NewDict()
	if body != nil {
		ret.Add(body, "webconsole")
	}
	appsrv.SendJSON(w, ret)
}

func handleAdbShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	serverId := env.Params["<id>"]
	serverDetails := compute_api.ServerDetails{}
	resp, err := modules.Servers.GetById(env.ClientSessin, serverId, nil)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	err = resp.Unmarshal(&serverDetails)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	phoneIp := ""
	adbPort := -1
	connStr, _ := env.Body.GetString("conn")
	if len(connStr) > 0 {
		parts := strings.Split(connStr, ":")
		if len(parts) > 0 {
			phoneIp = parts[0]
			if len(parts) > 1 {
				adbPort, _ = strconv.Atoi(parts[1])
			}
			if adbPort == 0 {
				adbPort = 5555
			}
		}
	}
	if len(phoneIp) == 0 {
		// fallback
		if len(serverDetails.Nics) > 0 {
			phoneIp = serverDetails.Nics[0].IpAddr
			portMaps := serverDetails.Nics[0].PortMappings
			for i := range portMaps {
				portMap := portMaps[i]
				if portMap.Port == 5555 && portMap.Protocol == "tcp" {
					adbPort = *portMap.HostPort
					break
				}
			}
			var errMsgs []string
			if !regutils.MatchIP4Addr(phoneIp) {
				errMsgs = append(errMsgs, "invalid phone IP")
			}
			if adbPort < 0 {
				errMsgs = append(errMsgs, "adb port not found")
			}
			if len(errMsgs) > 0 {
				httperrors.GeneralServerError(ctx, w, httperrors.NewNotSupportedError(strings.Join(errMsgs, ";")))
				return
			}
		} else {
			httperrors.GeneralServerError(ctx, w, httperrors.NewNotSupportedError("porting_mapping not supported"))
			return
		}
		{
			// test phoneIP is accessible
			err := netutils2.TestTcpPort(phoneIp, 5555, 3, 3)
			if err != nil {
				log.Errorf("TestTcpPort %s:%d fail %s", phoneIp, 5555, err)
				phoneIp = serverDetails.HostEIP
				if len(phoneIp) == 0 {
					phoneIp = serverDetails.HostAccessIp
				}
			} else {
				adbPort = 5555
			}
		}
	}

	info := command.SAdbShellInfo{
		HostIp:   phoneIp,
		HostPort: adbPort,
	}

	cmd, err := command.NewAdbShellCommand(&info, env.ClientSessin)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	handleCommandSession(ctx, cmd, w)
}
