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
	"bytes"
)

// Module represents name and args of ansible module to execute
type Module struct {
	// Name is ansible module name
	Name string
	// Args is a list of module arguments in form of key=value
	Args []string
}

// Host represents an ansible host
type Host struct {
	// Name denotes the host to operate on
	Name string
	// Vars is a map recording host vars
	Vars map[string]string
}

// GetVar returns variable value.  Second return value will be false if the
// variable does not exist
func (h *Host) GetVar(k string) (v string, exist bool) {
	if len(h.Vars) == 0 {
		return
	}
	v, exist = h.Vars[k]
	return
}

// SetVar sets variable k to value v
func (h *Host) SetVar(k, v string) {
	if h.Vars == nil {
		h.Vars = map[string]string{
			k: v,
		}
		return
	}
	h.Vars[k] = v
}

// Inventory contains a list of ansible hosts
type Inventory struct {
	Hosts []Host
}

// IsEmpty returns true if the inventory is empty
func (i *Inventory) IsEmpty() bool {
	return len(i.Hosts) == 0
}

// Data returns serialized form of the inventory as represented on disk
func (i *Inventory) Data() []byte {
	b := &bytes.Buffer{}
	for _, h := range i.Hosts {
		b.WriteString(h.Name)
		for k, v := range h.Vars {
			b.WriteRune(' ')
			b.WriteString(k)
			b.WriteRune('=')
			b.WriteString(v)
		}
		b.WriteRune('\n')
	}
	return b.Bytes()
}
