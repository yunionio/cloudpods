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

import "yunion.io/x/onecloud/pkg/apis"

type KubeNodePoolListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput

	CloudKubeClusterId string `json:"cloud_kube_cluster_id"`
}

type KubeNodePoolCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	NetworkIds    SKubeNetworkIds `json:"network_ids"`
	InstanceTypes SInstanceTypes  `json:"instance_types"`

	// default: 2
	MinInstanceCount int `json:"min_instance_count"`
	// default: 2
	MaxInstanceCount int `json:"max_instance_count"`
	// 预期节点数量, 不得小于min_instance_count
	// default: 2
	DesiredInstanceCount int `json:"desired_instance_count"`

	RootDiskSizeGb int `json:"root_disk_size_gb"`

	CloudKubeClusterId string `json:"cloud_kube_cluster_id"`

	// 秘钥id，若不传，则使用系统级秘钥
	KeypairId string `json:"keypair_id"`

	// swagger: ignore
	PublicKey string `json:"public_key"`
}

type KubeNodePoolDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.InfrasResourceBaseDetails
	DomainId string
}

type KubeNodePoolUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput
}
