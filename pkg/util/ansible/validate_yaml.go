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
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-yaml/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var controllerCallRe = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])(lookup|query)\s*\(`)

func ValidatePlaybookYAML(doc string) error {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return errors.Errorf("empty playbook")
	}
	if err := rejectControllerCalls(doc); err != nil {
		return err
	}
	var parsed interface{}
	if err := yaml.Unmarshal([]byte(doc), &parsed); err != nil {
		return errors.Wrap(err, "parse playbook")
	}
	return walkYAML(parsed, "")
}

func ValidateInventoryYAML(doc string) error {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return errors.Errorf("empty inventory")
	}
	if err := rejectControllerCalls(doc); err != nil {
		return err
	}
	var parsed interface{}
	if err := yaml.Unmarshal([]byte(doc), &parsed); err != nil {
		return errors.Wrap(err, "parse inventory")
	}
	return walkYAML(parsed, "")
}

func ValidateFilesJSON(files string) error {
	files = strings.TrimSpace(files)
	if files == "" {
		return nil
	}
	obj, err := jsonutils.ParseString(files)
	if err != nil {
		return errors.Wrap(err, "parse files")
	}
	m, err := obj.GetMap()
	if err != nil {
		return errors.Wrap(err, "files must be a json object")
	}
	for name := range m {
		if err := ValidatePlaybookFileName(name); err != nil {
			return err
		}
	}
	return nil
}

func rejectControllerCalls(doc string) error {
	if controllerCallRe.MatchString(doc) {
		return errors.Errorf("lookup/query is not allowed")
	}
	return nil
}

func walkYAML(v interface{}, parent string) error {
	switch t := v.(type) {
	case map[interface{}]interface{}:
		for k, val := range t {
			key := fmt.Sprint(k)
			if err := checkYAMLPair(parent, key, val); err != nil {
				return err
			}
			if err := walkYAML(val, key); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for key, val := range t {
			if err := checkYAMLPair(parent, key, val); err != nil {
				return err
			}
			if err := walkYAML(val, key); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range t {
			if s, ok := yamlString(item); ok && (parent == "hosts" || parent == "delegate_to") {
				if hostsContainLocal(s) {
					return errors.Errorf("%s %q is not allowed", parent, s)
				}
			}
			if err := walkYAML(item, parent); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkYAMLPair(parent, key string, val interface{}) error {
	lk := strings.ToLower(strings.TrimSpace(key))
	s, isStr := yamlString(val)
	switch lk {
	case "connection", "ansible_connection":
		if !isStr {
			return errors.Errorf("%s must be a string", key)
		}
		if _, ok := allowedConnections[strings.ToLower(strings.TrimSpace(s))]; !ok {
			return errors.Errorf("%s %q is not allowed", key, s)
		}
	case "delegate_to", "ansible_host":
		if isStr && hostsContainLocal(s) {
			return errors.Errorf("%s %q is not allowed", key, s)
		}
	case "hosts":
		if isStr && hostsContainLocal(s) {
			return errors.Errorf("hosts %q is not allowed", s)
		}
	case "local_action":
		return errors.Errorf("local_action is not allowed")
	case "ansible_user":
		if isStr && s != "" && !IsValidAnsibleUser(s) {
			return errors.Errorf("invalid ansible_user")
		}
	case "ansible_ssh_private_key_file":
		if isStr {
			if _, err := fileutils2.CleanRelSubpath(s); err != nil {
				return errors.Wrap(err, "ansible_ssh_private_key_file")
			}
		}
	case "import_playbook", "include", "include_playbook", "import_tasks", "include_tasks", "include_vars":
		if isStr {
			if strings.Contains(s, "://") || filepath.IsAbs(s) || strings.Contains(s, "{{") {
				return errors.Errorf("%s %q is not allowed", key, s)
			}
			if _, err := fileutils2.CleanRelSubpath(s); err != nil {
				return errors.Wrapf(err, "%s", key)
			}
		}
	}
	if isDeniedAnsibleVar(lk) {
		return errors.Errorf("%s is not allowed", key)
	}
	if parent == "hosts" && isLocalTarget(key) {
		return errors.Errorf("host %q is not allowed", key)
	}
	return nil
}

func hostsContainLocal(s string) bool {
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		host := stripHostPort(part)
		if isLocalTarget(host) {
			return true
		}
	}
	return false
}

func stripHostPort(part string) string {
	if strings.HasPrefix(part, "[") {
		end := strings.Index(part, "]")
		if end > 1 {
			return part[1:end]
		}
		return part
	}
	if i := strings.LastIndex(part, ":"); i > 0 {
		if net.ParseIP(part) == nil {
			return part[:i]
		}
	}
	return part
}

func yamlString(v interface{}) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case []byte:
		return string(t), true
	default:
		return "", false
	}
}
