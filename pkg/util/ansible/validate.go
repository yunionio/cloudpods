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
	"net"
	"path"
	"regexp"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var (
	ansibleUserRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	ansibleVarRe  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

var allowedConnections = map[string]struct{}{
	"":         {},
	"ssh":      {},
	"smart":    {},
	"paramiko": {},
	"winrm":    {},
	"psrp":     {},
}

var deniedAnsibleVars = map[string]struct{}{
	"ansible_shell_executable":   {},
	"ansible_python_interpreter": {},
	"ansible_ssh_executable":     {},
	"ansible_remote_tmp":         {},
	"ansible_executable":         {},
	"ansible_shell_type":         {},
	"ansible_become_exe":         {},
	"ansible_become_flags":       {},
	"ansible_ssh_args":           {},
	"ansible_async_dir":          {},
}

func IsValidAnsibleUser(user string) bool {
	return ansibleUserRe.MatchString(user)
}

func ValidateInventoryHostName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.Errorf("empty host name")
	}
	if strings.ContainsAny(name, " \t\r\n=") {
		return errors.Errorf("invalid host name")
	}
	if isLocalTarget(name) {
		return errors.Errorf("host %q is not allowed", name)
	}
	return nil
}

func ValidateInventoryVar(key, value string) error {
	if !ansibleVarRe.MatchString(key) {
		return errors.Errorf("invalid inventory variable %q", key)
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return errors.Errorf("invalid inventory variable %s", key)
	}
	lk := strings.ToLower(key)
	if isDeniedAnsibleVar(lk) {
		return errors.Errorf("inventory variable %s is not allowed", key)
	}
	switch lk {
	case "ansible_connection":
		if _, ok := allowedConnections[strings.ToLower(strings.TrimSpace(value))]; !ok {
			return errors.Errorf("ansible_connection %q is not allowed", value)
		}
	case "ansible_host":
		if isLocalTarget(value) {
			return errors.Errorf("ansible_host %q is not allowed", value)
		}
	case "ansible_user":
		if value != "" && !IsValidAnsibleUser(value) {
			return errors.Errorf("invalid ansible_user")
		}
	case "ansible_ssh_private_key_file":
		if _, err := fileutils2.CleanRelSubpath(value); err != nil {
			return errors.Wrap(err, "ansible_ssh_private_key_file")
		}
	}
	return nil
}

func ValidateInventoryHost(h Host) error {
	if err := ValidateInventoryHostName(h.Name); err != nil {
		return err
	}
	for k, v := range h.Vars {
		if err := ValidateInventoryVar(k, v); err != nil {
			return err
		}
	}
	return nil
}

func ValidatePlaybookFileName(name string) error {
	rel, err := fileutils2.CleanRelSubpath(name)
	if err != nil {
		return errors.Wrapf(err, "playbook file %q", name)
	}
	base := path.Base(rel)
	if base == "ansible.cfg" || base == ".ansible.cfg" {
		return errors.Errorf("playbook file %q is not allowed", name)
	}
	return nil
}

func ValidatePlaybook(pb *Playbook) error {
	if pb == nil {
		return errors.Errorf("empty playbook")
	}
	for i := range pb.Inventory.Hosts {
		if err := ValidateInventoryHost(pb.Inventory.Hosts[i]); err != nil {
			return err
		}
	}
	for name := range pb.Files {
		if err := ValidatePlaybookFileName(name); err != nil {
			return err
		}
	}
	return nil
}

func isDeniedAnsibleVar(key string) bool {
	if _, ok := deniedAnsibleVars[key]; ok {
		return true
	}
	if !strings.HasPrefix(key, "ansible_") {
		return false
	}
	for _, suffix := range []string{"_executable", "_interpreter", "_extra_args", "_common_args"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

func isLocalTarget(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false
	}
	switch s {
	case "localhost", "localhost.localdomain", "ip6-localhost", "ip6-loopback":
		return true
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified()
}
