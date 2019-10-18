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

package models

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/ansiblev2"
)

func (router *SRouter) ansibleHost() (*ansiblev2.Host, error) {
	vars := map[string]interface{}{
		"ansible_user": router.User,
		"ansible_host": router.Host,
	}
	if router.User != "root" {
		vars["ansible_become"] = "yes"
	}
	if router.PrivateKey != "" {
		vars["ansible_ssh_private_key_file"] = ".id_rsa"
	}
	if router.RealizeWgIfaces {
		if err := router.inventoryWireguardVars(vars); err != nil {
			return nil, err
		}
	}
	h := ansiblev2.NewHost()
	h.Vars = vars
	return h, nil
}

func (router *SRouter) playFiles() map[string]string {
	r := map[string]string{
		"wgX.conf.j2": wgX_conf_j2,
	}
	if router.PrivateKey != "" {
		r[".id_rsa"] = router.PrivateKey
	}
	return r
}

func (router *SRouter) playFilesStr() string {
	files := router.playFiles()
	r, _ := json.Marshal(files)
	return string(r)
}

func (router *SRouter) inventoryWireguardVars(vars map[string]interface{}) error {
	type (
		WgNetworks  []string
		WgInterface map[string]interface{}
		WgPeer      map[string]interface{}
		WgPeers     map[string]WgPeer
	)
	ifaces, err := IfaceManager.getByRouter(router)
	if err != nil {
		return err
	}
	wgnetworks := WgNetworks{}
	for i := range ifaces {
		iface := &ifaces[i]
		if iface.PrivateKey == "" {
			continue
		}
		ifacePeers, err := IfacePeerManager.getByIface(iface)
		if err != nil {
			return err
		}
		wgpeers := WgPeers{}
		for j := range ifacePeers {
			ifacePeer := &ifacePeers[j]
			if ifacePeer.PublicKey == "" {
				continue
			}
			wgpeer := WgPeer{
				"public_key":  ifacePeer.PublicKey,
				"allowed_ips": ifacePeer.AllowedIPs,
				"endpoint":    ifacePeer.Endpoint,
			}
			if ifacePeer.PersistentKeepalive > 0 {
				wgpeer["persistent_keepalive"] = ifacePeer.PersistentKeepalive
			}
			wgpeers[ifacePeer.Name] = wgpeer
		}
		if len(wgpeers) == 0 {
			continue
		}
		vars["wireguard_"+iface.Ifname+"_interface"] = WgInterface{
			"private_key": iface.PrivateKey,
			"listen_port": iface.ListenPort,
		}
		vars["wireguard_"+iface.Ifname+"_peers"] = wgpeers
		wgnetworks = append(wgnetworks, iface.Ifname)
	}
	vars["wireguard_networks"] = wgnetworks
	return nil
}

func (router *SRouter) playInstallWireguard() *ansiblev2.Play {
	play := ansiblev2.NewPlay(
		&ansiblev2.Task{
			Name:       "Enable EPEL",
			ModuleName: "package",
			ModuleArgs: map[string]interface{}{
				"name":  "epel-release",
				"state": "present",
			},
		},
		&ansiblev2.Task{
			Name:       "Check existence of wireguard repo file",
			ModuleName: "stat",
			ModuleArgs: map[string]interface{}{
				"path": "/etc/yum.repos.d/_copr_jdoss-wireguard.repo",
			},
			Register: "wireguard_repo",
		},
		&ansiblev2.Task{
			Name:       "Enable wireguard repo from copr",
			ModuleName: "get_url",
			ModuleArgs: map[string]interface{}{
				"dest": "/etc/yum.repos.d/_copr_jdoss-wireguard.repo",
				"url":  "https://copr.fedorainfracloud.org/coprs/jdoss/wireguard/repo/epel-7/jdoss-wireguard-epel-7.repo",
			},
			When: "(not wireguard_repo.stat.exists) or (wireguard_repo.stat.size < 10)",
		},
		&ansiblev2.Task{
			Name:       "Install wireguard packages",
			ModuleName: "package",
			ModuleArgs: map[string]interface{}{
				"name":  "{{ item }}",
				"state": "present",
			},
			WithPlugin:    "items",
			WithPluginVal: []string{"wireguard-dkms", "wireguard-tools"},
		},
		&ansiblev2.Task{
			Name:       "Create /etc/wireguard",
			ModuleName: "file",
			ModuleArgs: map[string]interface{}{
				"path":  "/etc/wireguard",
				"state": "directory",
				"owner": "root",
				"group": "root",
			},
		},
	)
	play.Hosts = "all"
	play.Name = "Install WireGuard"
	return play
}

func (router *SRouter) playDeployWireguardNetworks() *ansiblev2.Play {
	play := ansiblev2.NewPlay(
		&ansiblev2.ShellTask{
			Name:         "List existing managed wireguard networks",
			Script:       `grep -rnl 'Ansible managed' /etc/wireguard/ | grep '\.conf$' | cut -d/ -f4 | cut -d. -f1`,
			Register:     "oldconfs",
			IgnoreErrors: true,
		},
		&ansiblev2.Task{
			Name:       "Backup stale wireguard confs",
			ModuleName: "copy",
			ModuleArgs: map[string]interface{}{
				"src":        "/etc/wireguard/{{ item }}.conf",
				"dest":       "/etc/wireguard/{{ item }}.conf.stale",
				"remote_src": "yes",
			},
			WithPlugin:    "items",
			WithPluginVal: "{{ oldconfs.stdout_lines }}",
			When:          "(not oldconfs.failed) and (item not in wireguard_networks)",
		},
		&ansiblev2.Task{
			Name:       "Remove stale wireguard confs",
			ModuleName: "file",
			ModuleArgs: map[string]interface{}{
				"path":  "/etc/wireguard/{{ item }}.conf",
				"state": "absent",
			},
			WithPlugin:    "items",
			WithPluginVal: "{{ oldconfs.stdout_lines }}",
			When:          "(not oldconfs.failed) and (item not in wireguard_networks)",
		},
		&ansiblev2.Task{
			Name:       "Disable stale wireguard networks",
			ModuleName: "service",
			ModuleArgs: map[string]interface{}{
				"name":    "wg-quick@{{ item }}",
				"state":   "stopped",
				"enabled": "no",
			},
			WithPlugin:    "items",
			WithPluginVal: "{{ oldconfs.stdout_lines }}",
			When:          "(not oldconfs.failed) and (item not in wireguard_networks)",
		},
		&ansiblev2.Task{
			Name:       "Configure wireguard conf",
			ModuleName: "template",
			ModuleArgs: map[string]interface{}{
				"src":  "wgX.conf.j2", // wgX_conf_j2
				"dest": "/etc/wireguard/{{ item }}.conf",
				"mode": 0600,
			},
			WithPlugin:    "items",
			WithPluginVal: "{{ wireguard_networks }}",
			Register:      "configuration",
		},
		&ansiblev2.Task{
			Name:       "Enable wg-quick@xx service",
			ModuleName: "service",
			ModuleArgs: map[string]interface{}{
				"name":    "wg-quick@{{ item }}",
				"enabled": "yes",
			},
			WithPlugin:    "items",
			WithPluginVal: "{{ wireguard_networks }}",
		},
		&ansiblev2.Task{
			Name:       "Restart wg-quick@xx service",
			ModuleName: "service",
			ModuleArgs: map[string]interface{}{
				"name":  "wg-quick@{{ item.1 }}",
				"state": "restarted",
			},
			WithPlugin:    "indexed_items",
			WithPluginVal: "{{ wireguard_networks }}",
			When:          "configuration.results[item.0].changed",
		},
	)
	play.Hosts = "all"
	play.Name = "Configure wireguard networks"
	return play
}

func (router *SRouter) playDeployRoutes() (*ansiblev2.Play, error) {
	r, err := RouteManager.routeLinesRouter(router)
	if err != nil {
		return nil, err
	}
	tasks := []ansiblev2.ITask{}
	i := 0
	for ifname, lines := range r {
		if len(lines) == 0 {
			continue
		}
		iface, err := IfaceManager.getByRouterIfname(router, ifname)
		if err != nil {
			return nil, errors.WithMessagef(err, "get iface %s", ifname)
		}
		filename := "route-" + ifname
		content := strings.Join(lines, "\n") + "\n"
		registerVar := fmt.Sprintf("var%d", i)
		i += 1
		tasks = append(tasks, &ansiblev2.Task{
			Name:       "Put routes for " + ifname,
			ModuleName: "copy",
			ModuleArgs: map[string]interface{}{
				"content": content,
				"dest":    "/etc/sysconfig/network-scripts/" + filename,
				"owner":   "root",
				"group":   "root",
				"mode":    "0644",
			},
			Register: registerVar,
		})
		if !iface.isTypeWireguard() {
			tasks = append(tasks, &ansiblev2.ShellTask{
				Name:         fmt.Sprintf("Apply routes (Ifup/ifdown %s)", ifname),
				Script:       fmt.Sprintf("ifdown %s; ifup %s", ifname, ifname),
				IgnoreErrors: true,
				When:         fmt.Sprintf("%s.changed", registerVar),
			})
		}
		// apply by diff on changed
	}
	play := ansiblev2.NewPlay(tasks...)
	play.Hosts = "all"
	play.Name = "Configure routes"
	return play, nil
}

func (router *SRouter) playDeployRules() (*ansiblev2.Play, error) {
	d, err := RuleManager.firewalldDirectByRouter(router)
	if err != nil {
		return nil, err
	}
	directXML, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	play := ansiblev2.NewPlay(
		&ansiblev2.Task{
			Name:       "Install firewalld",
			ModuleName: "package",
			ModuleArgs: map[string]interface{}{
				"name":  "firewalld",
				"state": "present",
			},
		},
		&ansiblev2.Task{
			Name:       "Enable firewalld",
			ModuleName: "service",
			ModuleArgs: map[string]interface{}{
				"name":    "firewalld",
				"enabled": "yes",
			},
		},
		&ansiblev2.Task{
			Name:       "Put firewalld direct.xml",
			ModuleName: "copy",
			ModuleArgs: map[string]interface{}{
				"content": string(directXML),
				"dest":    "/etc/firewalld/direct.xml",
				"owner":   "root",
				"group":   "root",
				"mode":    "0644",
			},
			Register: "direct_xml",
		},
		&ansiblev2.Task{
			Name:       "Restart firewalld",
			ModuleName: "service",
			ModuleArgs: map[string]interface{}{
				"name":  "firewalld",
				"state": "restarted",
			},
			When: "direct_xml.changed",
		},
	)
	play.Hosts = "all"
	play.Name = "Configure firewall rules"
	return play, nil
}

func (router *SRouter) playEssential() *ansiblev2.Play {
	play := ansiblev2.NewPlay(
		&ansiblev2.Task{
			Name:       "Enable ip_forward",
			ModuleName: "sysctl",
			ModuleArgs: map[string]interface{}{
				"name":   "net.ipv4.ip_forward",
				"value":  "1",
				"state":  "present",
				"reload": "yes",
			},
		},
	)
	play.Hosts = "all"
	play.Name = "Perform essential steps"
	return play
}
