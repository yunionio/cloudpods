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

package apis

// ContainerProbeHandlerExecAction describes a "run in container" action.
type ContainerProbeHandlerExecAction struct {
	// Command is the command line to execute inside the container, the working directory for the
	// command  is root ('/') in the container's filesystem. The command is simply exec'd, it is
	// not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use
	// a shell, you need to explicitly call out to that shell.
	// Exit status of 0 is treated as live/healthy and non-zero is unhealthy.
	// +optional
	Command []string `json:"command,omitempty"`
}

// URIScheme identifies the scheme used for connection to a host for Get actions
type URIScheme string

const (
	// URISchemeHTTP means that the scheme used will be http://
	URISchemeHTTP URIScheme = "HTTP"
	// URISchemeHTTPS means that the scheme used will be https://
	URISchemeHTTPS URIScheme = "HTTPS"
)

// HTTPHeader describes a custom header to be used in HTTP probes
type HTTPHeader struct {
	// The header field name
	Name string `json:"name"`
	// The header field value
	Value string `json:"value"`
}

// ContainerProbeHTTPGetAction describes an action based on HTTP Get requests.
type ContainerProbeHTTPGetAction struct {
	// Path to access on the HTTP server.
	// +optional
	Path string `json:"path,omitempty"`
	// Name or number of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port int `json:"port"`
	// Host name to connect to, defaults to the pod IP. You probably want to set
	// "Host" in httpHeaders instead.
	// +optional
	Host string `json:"host,omitempty"`
	// Scheme to use for connecting to the host.
	// Defaults to HTTP.
	// +optional
	Scheme URIScheme `json:"scheme,omitempty"`
	// Custom headers to set in the request. HTTP allows repeated headers.
	// +optional
	HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty"`
}

// ContainerProbeTCPSocketAction describes an action based on opening a socket
type ContainerProbeTCPSocketAction struct {
	// Number or name of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port int `json:"port"`
	// Optional: Host name to connect to, defaults to the pod IP.
	// +optional
	Host string `json:"host,omitempty"`
}

type ContainerProbeType string

const (
	ContainerProbeTypeLiveness  ContainerProbeType = "Liveness"
	ContainerProbeTypeReadiness ContainerProbeType = "Readiness"
	ContainerProbeTypeStartup   ContainerProbeType = "Startup"
)

// ContainerProbeHandler defines a specific action that should be taken
type ContainerProbeHandler struct {
	// One and only one of the following should be specified.
	// Exec specifies the action to take.
	Exec *ContainerProbeHandlerExecAction `json:"exec,omitempty"`
	// HTTPGet specifies the http request to perform.
	HTTPGet *ContainerProbeHTTPGetAction `json:"http_get,omitempty"`
	// TCPSocket specifies an action involving a TCP port.
	TCPSocket *ContainerProbeTCPSocketAction `json:"tcp_socket,omitempty"`
}

// ContainerProbe describes a health check to be performed against a container to determine whether it is
// alive or ready to receive traffic.
type ContainerProbe struct {
	// The action taken to determine the health of a container
	ContainerProbeHandler `json:",inline"`
	// Number of seconds after the container has started before liveness probes are initiated.
	// InitialDelaySeconds int32 `json:"initial_delay_seconds,omitempty"`
	// Number of seconds after which the probe times out.
	TimeoutSeconds int32 `json:"timeout_seconds,omitempty"`
	// How often (in seconds) to perform the probe.
	// Default to 10 seconds. Minimum value is 1.
	PeriodSeconds int32 `json:"period_seconds,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.
	SuccessThreshold int32 `json:"success_threshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Defaults to 3. Minimum value is 1.
	FailureThreshold int32 `json:"failure_threshold,omitempty"`
}
