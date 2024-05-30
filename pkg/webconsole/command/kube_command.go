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

package command

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	webconsole_api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/mcclient"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type K8sEnv struct {
	Session    *mcclient.ClientSession
	Cluster    string
	Namespace  string
	Pod        string
	Container  string
	Kubeconfig string
	Data       jsonutils.JSONObject
}

type Kubectl struct {
	*BaseCommand
	kubeconfig string
	// For display info
	InstanceName string
	IPs          []string
}

func NewKubectlCommand(s *mcclient.ClientSession, kubeconfig, namespace string) *Kubectl {
	name := o.Options.KubectlPath
	if len(namespace) == 0 {
		namespace = "default"
	}
	cmd := NewBaseCommand(s, name, "--namespace", namespace)
	return &Kubectl{
		BaseCommand: cmd,
		kubeconfig:  kubeconfig,
	}
}

func (c *Kubectl) SetInstanceName(name string) {
	c.InstanceName = name
}

func (c *Kubectl) GetInstanceName() string {
	return c.InstanceName
}

func (c *Kubectl) SetIPs(ips []string) {
	c.IPs = ips
}

func (c *Kubectl) GetIPs() []string {
	return c.IPs
}

func (c *Kubectl) GetCommand() *exec.Cmd {
	cmd := c.BaseCommand.GetCommand()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", c.kubeconfig))
	return cmd
}

func (c Kubectl) GetProtocol() string {
	return PROTOCOL_TTY
}

func (c *Kubectl) Cleanup() error {
	log.Debugf("Remove temp kubeconfig file: %s", c.kubeconfig)
	return os.Remove(c.kubeconfig)
}

type KubectlExec struct {
	*Kubectl
}

func (c *Kubectl) Exec() *KubectlExec {
	// Execute a command in a container
	cmd := &KubectlExec{
		Kubectl: c,
	}
	cmd.AppendArgs("exec")
	return cmd
}

func (c *KubectlExec) Stdin() *KubectlExec {
	// -i: Pass stdin to the container
	c.AppendArgs("-i")
	return c
}

func (c *KubectlExec) TTY() *KubectlExec {
	// -t: Stdin is a TTY
	c.AppendArgs("-t")
	return c
}

func (c *KubectlExec) Container(name string) *KubectlExec {
	if len(name) == 0 {
		return c
	}
	// -c: Container name. If ommitted, the first container in the pod will be chosen
	c.AppendArgs("-c", name)
	return c
}

func (c *KubectlExec) Pod(name string) *KubectlExec {
	// Pod name
	c.AppendArgs(name)
	return c
}

func (c *KubectlExec) Command(cmd string, args ...string) *KubectlExec {
	c.AppendArgs("--", cmd)
	c.AppendArgs(args...)
	return c
}

func NewPodBashCommand(env *K8sEnv) ICommand {
	shellRequest := webconsole_api.SK8sShellRequest{}
	err := env.Data.Unmarshal(&shellRequest)
	if err != nil {
		log.Errorf("env.Data.Unmarshal SK8sShellRequest: %s", err)
	}
	if shellRequest.Command == "" {
		shellRequest.Command = "sh"
	}
	args := make([]string, 0)
	if len(shellRequest.Env) > 0 {
		for k, v := range shellRequest.Env {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}
		args = append(args, shellRequest.Command)
		shellRequest.Command = "env"
	}
	args = append(args, shellRequest.Args...)

	kExec := NewKubectlCommand(env.Session, env.Kubeconfig, env.Namespace).Exec().
		Stdin().
		TTY().
		Pod(env.Pod).
		Container(env.Container).
		Command(shellRequest.Command, args...)
	if shellRequest.DisplayInfo != nil {
		kExec.SetInstanceName(shellRequest.DisplayInfo.InstanceName)
		kExec.SetIPs(shellRequest.DisplayInfo.IPs)
	}
	return kExec
}

type KubectlLog struct {
	*Kubectl
}

func (c *Kubectl) Logs() *KubectlLog {
	// Print the logs for a container in a pod
	cmd := &KubectlLog{
		Kubectl: c,
	}
	cmd.AppendArgs("logs")
	return cmd
}

func (c *KubectlLog) Follow() *KubectlLog {
	// -f: Specify if the logs should be streamed
	c.AppendArgs("-f")
	return c
}

func (c *KubectlLog) Pod(name string) *KubectlLog {
	// Pod name
	c.AppendArgs(name)
	return c
}

func (c *KubectlLog) Container(name string) *KubectlLog {
	if name == "" {
		return c
	}
	// -c, --container='': Print the logs of this container
	c.AppendArgs("-c", name)
	return c
}

func (c *KubectlLog) Since(data jsonutils.JSONObject) *KubectlLog {
	durationStr, _ := data.GetString("since")
	if durationStr == "" {
		return c
	}
	// --since: Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used
	if _, err := time.ParseDuration(durationStr); err != nil {
		log.Errorf("Failed to parse log since opt: %v", err)
		return c
	}
	c.AppendArgs("--since", durationStr)
	return c
}

func NewPodLogCommand(env *K8sEnv) ICommand {
	return NewKubectlCommand(env.Session, env.Kubeconfig, env.Namespace).Logs().
		Follow().
		Pod(env.Pod).
		Since(env.Data).
		Container(env.Container)
}
