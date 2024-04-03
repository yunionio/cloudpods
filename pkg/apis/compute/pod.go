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
	POD_STATUS_CREATING_CONTAINER              = "creating_container"
	POD_STATUS_CREATE_CONTAINER_FAILED         = "create_container_failed"
	POD_STATUS_STARTING_CONTAINER              = "starting_container"
	POD_STATUS_START_CONTAINER_FAILED          = "start_container_failed"
	POD_STATUS_STOPPING_CONTAINER              = "stopping_container"
	POD_STATUS_STOP_CONTAINER_FAILED           = "stop_container_failed"
	POD_STATUS_DELETING_CONTAINER              = "deleting_container"
	POD_STATUS_DELETE_CONTAINER_FAILED         = "delete_container_failed"
	POD_STATUS_SYNCING_CONTAINER_STATUS        = "syncing_container_status"
	POD_STATUS_SYNCING_CONTAINER_STATUS_FAILED = "sync_container_status_failed"
)

const (
	POD_METADATA_CRI_ID        = "cri_id"
	POD_METADATA_CRI_CONFIG    = "cri_config"
	POD_METADATA_PORT_MAPPINGS = "port_mappings"
)

type PodContainerCreateInput struct {
	// Container name
	Name string `json:"name"`
	ContainerSpec
}

type PodPortMappingProtocol string

const (
	PodPortMappingProtocolTCP = "tcp"
	PodPortMappingProtocolUDP = "udp"
	//PodPortMappingProtocolSCTP = "sctp"
)

type PodPortMappingPortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type PodPortMapping struct {
	Protocol      PodPortMappingProtocol   `json:"protocol"`
	ContainerPort int                      `json:"container_port"`
	HostPort      *int                     `json:"host_port,omitempty"`
	HostIp        string                   `json:"host_ip"`
	HostPortRange *PodPortMappingPortRange `json:"host_port_range,omitempty"`
}

type PodCreateInput struct {
	Containers   []*PodContainerCreateInput `json:"containers"`
	PortMappings []*PodPortMapping          `json:"port_mappings"`
}

type PodStartResponse struct {
	CRIId     string `json:"cri_id"`
	IsRunning bool   `json:"is_running"`
}

type PodMetadataPortMapping struct {
	Protocol      PodPortMappingProtocol `json:"protocol"`
	ContainerPort int32                  `json:"container_port"`
	HostPort      int32                  `json:"host_port,omitempty"`
	HostIp        string                 `json:"host_ip"`
}
