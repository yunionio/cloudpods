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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LoadbalancerListenerManager struct {
	modulebase.ResourceManager
}

var (
	LoadbalancerListeners LoadbalancerListenerManager
)

func init() {
	LoadbalancerListeners = LoadbalancerListenerManager{
		modules.NewComputeManager(
			"loadbalancerlistener",
			"loadbalancerlisteners",
			[]string{
				"id",
				"name",
				"loadbalancer_id",
				"status",
				"listener_type",
				"listener_port",
				"backend_port",
				"egress_mbps",
				"acl_status",
				"acl_type",
			},
			[]string{"tenant"},
		),
	}
	modules.RegisterCompute(&LoadbalancerListeners)
}
