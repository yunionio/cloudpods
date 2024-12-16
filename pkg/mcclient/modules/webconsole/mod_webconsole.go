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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	webconsole_api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	compute_options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

var (
	WebConsole WebConsoleManager
)

func init() {
	WebConsole = WebConsoleManager{NewWebConsoleManager()}

	modulebase.Register(&WebConsole)
}

type WebConsoleManager struct {
	modulebase.ResourceManager
}

func NewWebConsoleManager() modulebase.ResourceManager {
	return modulebase.ResourceManager{BaseManager: *modulebase.NewBaseManager("webconsole", "", "", nil, nil),
		Keyword: "webconsole", KeywordPlural: "webconsole"}
}

func (m WebConsoleManager) DoConnect(s *mcclient.ClientSession, connType, id, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(connType) == 0 {
		return nil, fmt.Errorf("Empty connection resource type")
	}
	url := fmt.Sprintf("/webconsole/%s", connType)
	if id != "" {
		url = fmt.Sprintf("%s/%s", url, id)
	}
	if action != "" {
		url = fmt.Sprintf("%s/%s", url, action)
	}
	return modulebase.Post(m.ResourceManager, s, url, params, "webconsole")
}

func (m WebConsoleManager) DoK8sConnect(s *mcclient.ClientSession, id, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "k8s", id, action, params)
}

func (m WebConsoleManager) DoK8sShellConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoK8sConnect(s, id, "shell", params)
}

func (m WebConsoleManager) DoAdbShell(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "adb", id, "shell", params)
}

func (m WebConsoleManager) doContainerAction(s *mcclient.ClientSession, data jsonutils.JSONObject, getArgs func(containerId string) []string) (jsonutils.JSONObject, error) {
	containerId, err := data.GetString("container_id")
	if err != nil {
		return nil, errors.Wrap(err, "get container_id")
	}
	if containerId == "" {
		return nil, httperrors.NewNotEmptyError("container_id")
	}
	obj, err := compute.Containers.Get(s, containerId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get container by %s", containerId)
	}
	containerName, _ := obj.GetString("name")
	serverId, _ := obj.GetString("guest_id")
	if serverId == "" {
		return nil, httperrors.NewNotFoundError("not found guest_id from container %s", obj)
	}
	serverObj, err := compute.Servers.Get(s, serverId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get server by %s", serverId)
	}
	serverDetails := &compute_api.ServerDetails{}
	if err := serverObj.Unmarshal(serverDetails); err != nil {
		return nil, errors.Wrapf(err, "unmarshal server details: %s", serverObj)
	}
	info := &webconsole_api.SK8sShellDisplayInfo{
		InstanceName: fmt.Sprintf("%s/%s", serverDetails.Name, containerName),
		IPs:          strings.Split(serverDetails.IPs, ","),
	}
	args := getArgs(containerId)
	return m.doCloudShell(s, info, "/opt/yunion/bin/climc", args...)
}

func (m WebConsoleManager) DoContainerExec(s *mcclient.ClientSession, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.doContainerAction(s, data, func(containerId string) []string {
		return []string{"container-exec", containerId, "sh"}
	})
}

func (m WebConsoleManager) DoContainerLog(s *mcclient.ClientSession, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	opt := new(compute_options.ContainerLogOptions)
	data.Unmarshal(opt)
	return m.doContainerAction(s, data, func(containerId string) []string {
		args := []string{"container-log"}
		if opt.Tail > 0 {
			args = append(args, "--tail", fmt.Sprintf("%d", opt.Tail))
		}
		if opt.LimitBytes > 0 {
			args = append(args, "--limit-bytes", fmt.Sprintf("%d", opt.LimitBytes))
		}
		if opt.Since != "" {
			args = append(args, "--since", opt.Since)
		}
		if opt.Follow {
			args = append(args, "-f")
		}
		args = append(args, containerId)
		return args
	})
}

type CloudShellRequest struct {
	InstanceName string   `json:"instance_name"`
	IPs          []string `json:"ips"`
}

func (m WebConsoleManager) DoCloudShell(s *mcclient.ClientSession, _ jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.doCloudSshShell(s, nil, "", nil, nil)
}

func (m WebConsoleManager) climcSshConnect(s *mcclient.ClientSession, hostname string, command string, args []string, env map[string]string, info *webconsole_api.SK8sShellDisplayInfo) (jsonutils.JSONObject, error) {
	if hostname == "" {
		hostname = "climc"
	}
	// maybe running in docker compose environment, so try to use ssh way
	data, err := m.DoClimcSshConnect(s, hostname, 22, command, args, env, info)
	if err != nil {
		return nil, errors.Wrap(err, "DoClimcSshConnect")
	}
	return data, nil
}

func (m WebConsoleManager) doActionWithClimcPod(
	s *mcclient.ClientSession,
	af func(s *mcclient.ClientSession, clusterId string, pod jsonutils.JSONObject) (jsonutils.JSONObject, error),
) (jsonutils.JSONObject, error) {
	adminSession := s
	if auth.IsAuthed() {
		adminSession = auth.GetAdminSession(s.GetContext(), s.GetRegion())
	}

	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONTrue, "system")
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString("system-default"), "name")
	clusters, err := k8s.KubeClusters.List(adminSession, query)
	if err != nil {
		return nil, errors.Wrap(err, "list k8s cluster")
	}
	clusterId, _ := clusters.Data[0].GetString("id")
	if len(clusterId) == 0 {
		return nil, httperrors.NewNotFoundError("cluster system-default no id")
	}
	query = jsonutils.NewDict()
	query.Add(jsonutils.NewString(clusterId), "cluster")
	query.Add(jsonutils.NewString("onecloud"), "namespace")
	query.Add(jsonutils.NewString("climc"), "search")
	query.Add(jsonutils.JSONTrue, "details")
	pods, err := k8s.Pods.List(adminSession, query)
	if err != nil {
		return nil, errors.Wrap(err, "Pods")
	}
	if len(pods.Data) == 0 {
		return nil, httperrors.NewNotFoundError("pod climc not found")
	}
	pod := pods.Data[0]
	return af(s, clusterId, pod)
}

func (m WebConsoleManager) doCloudShell(s *mcclient.ClientSession, info *webconsole_api.SK8sShellDisplayInfo, cmd string, args ...string) (jsonutils.JSONObject, error) {
	endpointType := "internal"
	authUrl, err := s.GetServiceURL("identity", endpointType)
	if err != nil {
		return nil, httperrors.NewNotFoundError("auth_url not found")
	}
	env := map[string]string{
		"OS_AUTH_TOKEN":           s.GetToken().GetTokenString(),
		"OS_PROJECT_NAME":         s.GetProjectName(),
		"OS_PROJECT_DOMAIN":       s.GetProjectDomain(),
		"OS_AUTH_URL":             authUrl,
		"OS_ENDPOINT_TYPE":        endpointType,
		"YUNION_USE_CACHED_TOKEN": "false",
		"YUNION_INSECURE":         "true",
		"OS_USERNAME":             "",
		"OS_PASSWORD":             "",
		"OS_DOMAIN_NAME":          "",
		"OS_ACCESS_KEY":           "",
		"OS_SECRET_KEY":           "",
		"OS_TRY_TERM_WIDTH":       "false",
	}
	return m.doCloudSshShell(s, info, cmd, args, env)
	/*return m.doActionWithClimcPod(s, func(s *mcclient.ClientSession, clusterId string, pod jsonutils.JSONObject) (jsonutils.JSONObject, error) {
		req := webconsole_api.SK8sShellRequest{}
		req.Cluster = clusterId
		req.Namespace = "onecloud"
		req.Container = "climc"
		req.Command = cmd
		req.Args = args
		endpointType := "internal"
		authUrl, err := s.GetServiceURL("identity", endpointType)
		if err != nil {
			return nil, httperrors.NewNotFoundError("auth_url not found")
		}
		req.Env = map[string]string{
			"OS_AUTH_TOKEN":           s.GetToken().GetTokenString(),
			"OS_PROJECT_NAME":         s.GetProjectName(),
			"OS_PROJECT_DOMAIN":       s.GetProjectDomain(),
			"OS_AUTH_URL":             authUrl,
			"OS_ENDPOINT_TYPE":        endpointType,
			"YUNION_USE_CACHED_TOKEN": "false",
			"YUNION_INSECURE":         "true",
			"OS_USERNAME":             "",
			"OS_PASSWORD":             "",
			"OS_DOMAIN_NAME":          "",
			"OS_ACCESS_KEY":           "",
			"OS_SECRET_KEY":           "",
			"OS_TRY_TERM_WIDTH":       "false",
		}
		req.DisplayInfo = info
		podName, err := pod.GetString("name")
		if err != nil {
			return nil, errors.Wrapf(err, "get pod name from: %s", pod.String())
		}
		return m.DoK8sConnect(s, podName, "shell", jsonutils.Marshal(req))
	})*/
}

func (m WebConsoleManager) doCloudSshShell(s *mcclient.ClientSession, info *webconsole_api.SK8sShellDisplayInfo, command string, args []string, env map[string]string) (jsonutils.JSONObject, error) {
	data, err := m.doActionWithClimcPod(s, func(s *mcclient.ClientSession, clusterId string, pod jsonutils.JSONObject) (jsonutils.JSONObject, error) {
		podIP, err := pod.GetString("podIP")
		if err != nil {
			return nil, errors.Wrap(err, "get podIP")
		}
		return m.climcSshConnect(s, podIP, command, args, env, info)
	})
	errs := []error{}
	if err != nil {
		errs = append(errs, err)
		// try climc ssh
		data, err := m.climcSshConnect(s, "", command, args, env, info)
		if err != nil {
			errs = append(errs, err)
			return nil, errors.NewAggregate(errs)
		}
		return data, nil
	}
	return data, nil

}

func (m WebConsoleManager) DoK8sLogConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoK8sConnect(s, id, "log", params)
}

func (m WebConsoleManager) DoBaremetalConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "baremetal", id, "", params)
}

func (m WebConsoleManager) DoSshConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "ssh", id, "", params)
}

func (m WebConsoleManager) DoServerConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "server", id, "", params)
}

func (m WebConsoleManager) DoServerRDPConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "server-rdp", id, "", params)
}

func (m WebConsoleManager) DoClimcSshConnect(s *mcclient.ClientSession, ip string, port int, command string, args []string, env map[string]string, info *webconsole_api.SK8sShellDisplayInfo) (jsonutils.JSONObject, error) {
	data := jsonutils.Marshal(map[string]interface{}{
		"username":      "root",
		"keep_username": true,
		"ip_addr":       ip,
		"port":          port,
		"name":          "climc",
		"env":           env,
		"command":       command,
		"args":          args,
		"display_info":  info,
	})
	body := jsonutils.NewDict()
	body.Set("webconsole", data)
	return m.DoConnect(s, "climc", "shell", "", body)
}
