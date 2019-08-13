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

package ansiblev2

import (
	"testing"
)

func TestPlaybookString(t *testing.T) {
	play := NewPlay(
		&Task{
			Name:       "Enable ip_forward",
			ModuleName: "sysctl",
			ModuleArgs: map[string]interface{}{
				"name":   "net.ipv4.ip_forward",
				"value":  "1",
				"state":  "present",
				"reload": "yes",
			},
		},
		&Task{
			Name:       "Enable EPEL",
			ModuleName: "package",
			ModuleArgs: map[string]interface{}{
				"name":  "epel-release",
				"state": "present",
			},
			When: `ansible_distribution != "Fedora"`,
		},
		&Task{
			Name:       "Install wireguard packages",
			ModuleName: "package",
			ModuleArgs: map[string]interface{}{
				"name":  "{{ item }}",
				"state": "present",
			},
			WithPlugin:    "items",
			WithPluginVal: []interface{}{"wireguard-dkms", "wireguard-tools"},
		},
		&Task{
			Name:       "Create /etc/wireguard",
			ModuleName: "file",
			ModuleArgs: map[string]interface{}{
				"path":  "/etc/wireguard",
				"staet": "directory",
				"owner": "root",
				"group": "root",
			},
		},
	)
	play.Hosts = "all"
	configureBlock := NewBlock(
		&Task{
			Name:       "Configure {{ item }}",
			ModuleName: "template",
			ModuleArgs: map[string]interface{}{
				"src":  "wgX.conf.j2", //XXX
				"dest": "/etc/wireguard/{{ item }}.conf",
				"mode": 0600,
			},
			Register: "configuration",
		},
		&Task{
			Name:       "Enable wg-quick@{{ item }} service",
			ModuleName: "service",
			ModuleArgs: map[string]interface{}{
				"name":    "wg-quick@{{ item }}",
				"state":   "started",
				"enabled": "yes",
			},
		},
		&Task{
			Name:       "Restart wg-quick@{{ item }} service",
			ModuleName: "service",
			ModuleArgs: map[string]interface{}{
				"name":  "wg-quick@{{ item }}",
				"state": "restarted",
			},
			When: "configuration is changed",
		},
	)
	configureBlock.Name = "Configure wireguard networks"
	play.Tasks = append(play.Tasks, configureBlock)
	pb := NewPlaybook(play)
	t.Logf("\n%s", pb.String())
}
