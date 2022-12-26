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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type WorkerManager struct {
	modulebase.ResourceManager
}

var (
	Workers WorkerManager
)

func init() {
	Workers = WorkerManager{modules.NewComputeManager("workers", "worker_stats",
		[]string{"name", "queue_cnt", "active_worker_cnt", "backlog", "detach_worker_cnt", "max_worker_cnt"},
		[]string{})}

	modules.RegisterCompute(&Workers)
}

func (this *WorkerManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error) {
	serviceType := apis.SERVICE_TYPE_REGION + "_v2"
	if params.Contains("service_type") {
		serviceType, _ = params.GetString("service_type")
	}
	return modulebase.ListWorkers(s, serviceType)
}
