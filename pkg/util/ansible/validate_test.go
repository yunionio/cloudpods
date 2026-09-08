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

package ansible

import (
	"strings"
	"testing"
)

func TestValidateInventoryHost(t *testing.T) {
	ok := Host{Name: "10.1.2.3", Vars: map[string]string{
		"ansible_user":     "cloudroot",
		"ansible_password": "p@ss word",
		"ansible_become":   "yes",
		"ansible_host":     "10.1.2.3",
		"ansible_port":     "22",
		"repo_base_url":    "https://example.com",
	}}
	if err := ValidateInventoryHost(ok); err != nil {
		t.Fatalf("valid host: %v", err)
	}
	if err := ValidateInventoryHost(Host{
		Name: "10.1.2.3",
		Vars: map[string]string{
			"ansible_connection":                   "winrm",
			"ansible_user":                         "Administrator",
			"ansible_password":                     "secret",
			"ansible_winrm_transport":              "ntlm",
			"ansible_winrm_server_cert_validation": "ignore",
		},
	}); err != nil {
		t.Fatalf("winrm host: %v", err)
	}

	if err := ValidateInventoryHost(Host{Name: "127.0.0.1"}); err == nil {
		t.Fatal("loopback host should fail")
	}
	if err := ValidateInventoryHost(Host{Name: "localhost"}); err == nil {
		t.Fatal("localhost should fail")
	}
	if err := ValidateInventoryHost(Host{
		Name: "10.1.2.3",
		Vars: map[string]string{"ansible_connection": "local"},
	}); err == nil {
		t.Fatal("local connection should fail")
	}
	if err := ValidateInventoryHost(Host{
		Name: "10.1.2.3",
		Vars: map[string]string{"ansible_host": "127.0.0.1"},
	}); err == nil {
		t.Fatal("loopback ansible_host should fail")
	}
	if err := ValidateInventoryHost(Host{
		Name: "10.1.2.3",
		Vars: map[string]string{"ansible_user": "cloudroot ansible_connection=local"},
	}); err == nil {
		t.Fatal("injected ansible_user should fail")
	}
	if err := ValidateInventoryHost(Host{
		Name: "10.1.2.3 cloudroot ansible_connection=local",
	}); err == nil {
		t.Fatal("injected host name should fail")
	}
	if err := ValidateInventoryHost(Host{
		Name: "10.1.2.3",
		Vars: map[string]string{"ansible_ssh_common_args": "-o ProxyCommand=id"},
	}); err == nil {
		t.Fatal("ssh extra args should fail")
	}
}

func TestValidatePlaybookFiles(t *testing.T) {
	pb := &Playbook{
		Inventory: Inventory{Hosts: []Host{{Name: "10.1.2.3"}}},
		Files: map[string][]byte{
			"../etc/passwd": []byte("x"),
		},
	}
	if err := ValidatePlaybook(pb); err == nil {
		t.Fatal("path escape should fail")
	}
	pb.Files = map[string][]byte{"ansible.cfg": []byte("x")}
	if err := ValidatePlaybook(pb); err == nil {
		t.Fatal("ansible.cfg should fail")
	}
	pb.Files = map[string][]byte{"a/b.txt": []byte("x")}
	if err := ValidatePlaybook(pb); err != nil {
		t.Fatalf("relative file: %v", err)
	}
	pb.Files = map[string][]byte{".id_rsa": []byte("x")}
	if err := ValidatePlaybook(pb); err != nil {
		t.Fatalf("dotfile: %v", err)
	}
}

func TestInventoryDataQuotesValues(t *testing.T) {
	inv := Inventory{Hosts: []Host{{
		Name: "10.1.2.3",
		Vars: map[string]string{
			"ansible_user":     "cloudroot",
			"ansible_password": "p@ss word ansible_connection=local",
		},
	}}}
	got := string(inv.Data())
	quoted := quoteInventoryValue("p@ss word ansible_connection=local")
	if !strings.Contains(got, quoted) {
		t.Fatalf("expected quoted password %s, got %q", quoted, got)
	}
	if strings.Contains(got, " ansible_connection=local") && !strings.Contains(got, quoted) {
		t.Fatalf("unquoted injection: %q", got)
	}
}

func TestValidatePlaybookYAML(t *testing.T) {
	if err := ValidatePlaybookYAML(`- hosts: all
  become: true
  tasks:
    - service:
        name: network
        state: restarted
`); err != nil {
		t.Fatalf("valid playbook: %v", err)
	}
	if err := ValidatePlaybookYAML(`- hosts: localhost
  connection: local
  tasks:
    - shell: id
`); err == nil {
		t.Fatal("local playbook should fail")
	}
	if err := ValidatePlaybookYAML(`- hosts: all
  tasks:
    - command: id
      delegate_to: localhost
`); err == nil {
		t.Fatal("delegate_to localhost should fail")
	}
	if err := ValidatePlaybookYAML(`- hosts: all,localhost
  tasks:
    - ping:
`); err == nil {
		t.Fatal("hosts list with localhost should fail")
	}
	if err := ValidatePlaybookYAML(`- hosts: all
  tasks:
    - debug:
        msg: "{{ lookup('pipe', 'id') }}"
`); err == nil {
		t.Fatal("lookup should fail")
	}
	if err := ValidatePlaybookYAML(`- hosts: all
  tasks:
    - local_action: shell id
`); err == nil {
		t.Fatal("local_action should fail")
	}
}

func TestValidateInventoryYAML(t *testing.T) {
	if err := ValidateInventoryYAML(`all:
  hosts:
    gw1:
      ansible_user: root
      ansible_host: 10.1.2.3
      ansible_ssh_private_key_file: .id_rsa
      ansible_become: yes
`); err != nil {
		t.Fatalf("valid inventory: %v", err)
	}
	if err := ValidateInventoryYAML(`all:
  hosts:
    x:
      ansible_connection: local
`); err == nil {
		t.Fatal("local inventory should fail")
	}
	if err := ValidateInventoryYAML(`all:
  hosts:
    localhost:
      ansible_user: root
`); err == nil {
		t.Fatal("localhost inventory host should fail")
	}
}

func TestValidateFilesJSON(t *testing.T) {
	if err := ValidateFilesJSON(`{".id_rsa":"k","wgX.conf.j2":"t"}`); err != nil {
		t.Fatalf("valid files: %v", err)
	}
	if err := ValidateFilesJSON(`{"../etc/passwd":"x"}`); err == nil {
		t.Fatal("path escape should fail")
	}
}
