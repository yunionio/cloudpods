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

package k8s

import (
	"yunion.io/x/jsonutils"
)

type MachineCreateOptions struct {
	CLUSTER  string `help:"Cluster id"`
	ROLE     string `help:"Machine role" choices:"node|controlplane"`
	Type     string `help:"Resource type" choices:"vm|baremetal" json:"resource_type"`
	Instance string `help:"VM or host instance id" json:"resource_id"`
	Name     string `help:"Name of node"`
}

func (o MachineCreateOptions) Params() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	params.Add(jsonutils.NewString(o.CLUSTER), "cluster")
	if o.ROLE != "" {
		params.Add(jsonutils.NewString(o.ROLE), "role")
	}
	if o.Instance != "" {
		params.Add(jsonutils.NewString(o.Instance), "resource_id")
	}
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "resource_type")
	}
	return params
}
