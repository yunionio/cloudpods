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

package compute

const (
	MethodDirect       = "direct"
	MethodEIP          = "eip"
	MethodDNAT         = "dnat"
	MethodProxyForward = "proxy_forward"
)

type GuestSshableMethodData struct {
	Method string `json:"method"`
	Host   string `json:"host"`
	Port   int    `json:"port"`

	Sshable bool   `json:"sshable"`
	Reason  string `json:"reason"`

	ForwardDetails ForwardDetails `json:"forward_details"`
}

type ForwardDetails struct {
	ProxyAgentId    string `json:"proxy_agent_id"`
	ProxyEndpointId string `json:"proxy_endpoint_id"`
}

type GuestSshableOutput struct {
	User      string `json:"user"`
	PublicKey string `json:"public_key"`

	MethodTried []GuestSshableMethodData `json:"method_tried"`
}

type GuestHaveAgentInput struct {
}

type GuestHaveAgentOutput struct {
	Have bool `json:"have"`
}

type GuestMakeSshableInput struct {
	User       string `json:"user"`
	PrivateKey string `json:"private_key"`
	Password   string `json:"password"`
	Port       int    `json:"port"`
}

type GuestMakeSshableOutput struct {
	AnsiblePlaybookId string `json:"ansible_playbook_id"`
}

type GuestMakeSshableCmdOutput struct {
	ShellCmd string `json:"shell_cmd"`
}

type GuestSetSshportInput struct {
	Port int `json:"port"`
}

type GuestSshportOutput struct {
	Port int `json:"port"`
}
