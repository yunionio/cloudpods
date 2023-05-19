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

const (
	KUBE_CLUSTER_STATUS_RUNNING  = "running"
	KUBE_CLUSTER_STATUS_CREATING = "creating"
	KUBE_CLUSTER_STATUS_DELETING = "deleting"
	KUBE_CLUSTER_STATUS_ABNORMAL = "abnormal"
	// 升级中
	KUBE_CLUSTER_STATUS_UPDATING = "updating"
	// 升级失败
	KUBE_CLUSTER_STATUS_UPDATING_FAILED = "updating_failed"
	// 伸缩中
	KUBE_CLUSTER_STATUS_SCALING = "scaling"
	// 停止
	KUBE_CLUSTER_STATUS_STOPPED = "stopped"
)
