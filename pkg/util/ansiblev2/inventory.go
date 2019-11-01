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
	"fmt"

	"github.com/go-yaml/yaml"
)

func newVars(args ...interface{}) map[string]interface{} {
	if len(args)&1 != 0 {
		panic("odd number of args for key/value pairs!")
	}
	vars := map[string]interface{}{}
	for i := 0; i < len(args); i += 2 {
		if k, ok := args[i].(string); !ok {
			panic(fmt.Sprintf("the %drd key is not string type: %#v", i, args[i]))
		} else {
			vars[k] = args[i+1]
		}
	}
	return vars
}

type Host struct {
	Vars map[string]interface{}
}

func NewHost(args ...interface{}) *Host {
	return &Host{
		Vars: newVars(args...),
	}
}

type HostGroup struct {
	Hosts    map[string]*Host
	Children map[string]*HostGroup
	Vars     map[string]interface{}
}

func NewHostGroup(args ...interface{}) *HostGroup {
	return &HostGroup{
		Vars: newVars(args...),
	}
}

func (hg *HostGroup) SetHost(name string, host *Host) *HostGroup {
	if hg.Hosts == nil {
		hg.Hosts = map[string]*Host{}
	}
	hg.Hosts[name] = host
	return hg
}

func (hg *HostGroup) SetChild(name string, child *HostGroup) *HostGroup {
	if hg.Children == nil {
		hg.Children = map[string]*HostGroup{}
	}
	hg.Children[name] = child
	return hg
}

func (hg *HostGroup) MarshalYAML() (interface{}, error) {
	hosts := map[string]interface{}{}
	for name := range hg.Hosts {
		hosts[name] = hg.Hosts[name].Vars
	}
	children := map[string]interface{}{}
	for name := range hg.Children {
		children[name], _ = hg.Children[name].MarshalYAML()
	}

	r := map[string]interface{}{}
	if len(hosts) > 0 {
		r["hosts"] = hosts
	}
	if len(children) > 0 {
		r["children"] = children
	}
	if len(hg.Vars) > 0 {
		r["vars"] = hg.Vars
	}
	return r, nil
}

type Inventory struct {
	HostGroup
}

func NewInventory(args ...interface{}) *Inventory {
	hg := NewHostGroup(args...)
	inv := &Inventory{
		HostGroup: *hg,
	}
	return inv
}

func (inv *Inventory) MarshalYAML() (interface{}, error) {
	all, _ := inv.HostGroup.MarshalYAML()
	r := map[string]interface{}{
		"all": all,
	}
	return r, nil
}

func (inv *Inventory) String() string {
	b, _ := yaml.Marshal(inv)
	return string(b)
}
