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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

// String implements gotypes.ISerializable
func (pb *Playbook) String() string {
	return jsonutils.Marshal(pb).String()
}

// IsZero implements gotypes.ISerializable
func (pb *Playbook) IsZero() bool {
	if len(pb.Inventory.Hosts) == 0 {
		return true
	}
	if len(pb.Modules) == 0 {
		return true
	}
	return false
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&Playbook{}), func() gotypes.ISerializable {
		return NewPlaybook()
	})
}
