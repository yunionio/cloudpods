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
	POD_STATUS_CREATING_CONTAINER      = "creating_container"
	POD_STATUS_CREATE_CONTAINER_FAILED = "create_container_failed"
	POD_STATUS_DELETING_CONTAINER      = "deleting_container"
	POD_STATUS_DELETE_CONTAINER_FAILED = "delete_container_failed"
)

const (
	POD_METADATA_CRI_ID     = "cri_id"
	POD_METADATA_CRI_CONFIG = "cri_config"
)

type PodContainerCreateInput struct {
	// Container name
	Name string `json:"name"`
	ContainerSpec
}

type PodPortMappingProtocol string

const (
	PodPortMappingProtocolTCP  = "tcp"
	PodPortMappingProtocolUDP  = "udp"
	PodPortMappingProtocolSCTP = "sctp"
)

type PodPortMapping struct {
	Protocol      PodPortMappingProtocol `json:"protocol"`
	ContainerPort int32                  `json:"container_port"`
	HostPort      int32                  `json:"host_port"`
	HostIp        string                 `json:"host_ip"`
}

type PodCreateInput struct {
	Containers   []*PodContainerCreateInput `json:"containers"`
	PortMappings []*PodPortMapping          `json:"port_mappings"`
}

type PodStartResponse struct {
	CRIId     string `json:"cri_id"`
	IsRunning bool   `json:"is_running"`
}
