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
