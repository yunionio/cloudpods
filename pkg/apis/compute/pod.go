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
	"time"

	"yunion.io/x/onecloud/pkg/httperrors"
)

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
	POD_STATUS_CRASH_LOOP_BACK_OFF             = "crash_loop_back_off"
	POD_STATUS_CONTAINER_EXITED                = "container_exited"
)

const (
	POD_METADATA_CRI_ID                   = "cri_id"
	POD_METADATA_CRI_CONFIG               = "cri_config"
	POD_METADATA_PORT_MAPPINGS            = "port_mappings"
	POD_METADATA_POST_STOP_CLEANUP_CONFIG = "post_stop_cleanup_config"
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

const (
	POD_PORT_MAPPING_RANGE_START = 20000
	POD_PORT_MAPPING_RANGE_END   = 25000
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

type PodSecurityContext struct {
	RunAsUser  *int64 `json:"run_as_user,omitempty"`
	RunAsGroup *int64 `json:"run_as_group,omitempty"`
}

type PodCreateInput struct {
	Containers []*PodContainerCreateInput `json:"containers"`
	HostIPC    bool                       `json:"host_ipc"`
	//PortMappings    []*PodPortMapping          `json:"port_mappings"`
	SecurityContext *PodSecurityContext `json:"security_context,omitempty"`
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

type GuestSetPortMappingsInput struct {
	PortMappings []*PodPortMapping `json:"port_mappings"`
}

type PodLogOptions struct {
	// The container for which to stream logs. Defaults to only container if there is one container in the pod.
	// +optional
	Container string `json:"container,omitempty"`
	// Follow the log stream of the pod. Defaults to false.
	// +optional
	Follow bool `json:"follow,omitempty"`
	// Return previous terminated container logs. Defaults to false.
	// +optional
	Previous bool `json:"previous,omitempty"`
	// A relative time in seconds before the current time from which to show logs. If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	// +optional
	SinceSeconds *int64 `json:"sinceSeconds,omitempty"`
	// An RFC3339 timestamp from which to show logs. If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	// +optional
	SinceTime *time.Time `json:"sinceTime,omitempty"`
	// If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line
	// of log output. Defaults to false.
	// +optional
	Timestamps bool `json:"timestamps,omitempty"`
	// If set, the number of lines from the end of the logs to show. If not specified,
	// logs are shown from the creation of the container or sinceSeconds or sinceTime
	// +optional
	TailLines *int64 `json:"tailLines,omitempty"`
	// If set, the number of bytes to read from the server before terminating the
	// log output. This may not display a complete final line of logging, and may return
	// slightly more or slightly less than the specified limit.
	// +optional
	LimitBytes *int64 `json:"limitBytes,omitempty"`

	// insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the
	// serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver
	// and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real
	// kubelet.  If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the
	// connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept
	// the actual log data coming from the real kubelet).
	// +optional
	InsecureSkipTLSVerifyBackend bool `json:"insecureSkipTLSVerifyBackend,omitempty"`
}

func ValidatePodLogOptions(opts *PodLogOptions) error {
	if opts.TailLines != nil && *opts.TailLines < 0 {
		return httperrors.NewInputParameterError("negative tail lines")
	}
	if opts.LimitBytes != nil && *opts.LimitBytes < 1 {
		return httperrors.NewInputParameterError("limit_bytes must be greater than zero")
	}
	if opts.SinceSeconds != nil && opts.SinceTime != nil {
		return httperrors.NewInputParameterError("at most one of since_time or since_seconds must be specified")
	}
	if opts.SinceSeconds != nil {
		if *opts.SinceSeconds < 1 {
			return httperrors.NewInputParameterError("since_seconds must be greater than zero")
		}
	}
	return nil
}

type PodPostStopCleanupConfig struct {
	Dirs []string `json:"dirs"`
}
