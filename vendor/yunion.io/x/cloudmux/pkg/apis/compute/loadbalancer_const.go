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

// Load balancer status transition (for spec status)
//
//	                create          start           stop            delete
//	init            running         -               -               -
//	running		-		-		stopped		stopped
//	stopped		-		running		-		-
//
// Each entity will have spec and runtime version.  Spec version will increment
// on entity attribute update.  Runtime version will be filled by the scheduler
// to the newest spec it has seen and committed
//
// When spec and runtime version differ, scheduler will set runtime version to
// "configuring", "stopping" and will finally transition to a terminal state.
//
// In the case of instance has PendingDeleted marked, it is also the
// scheduler's duty to make the runtime status to stopped and finally the
// entity in question
const (
	LB_STATUS_ENABLED  = "enabled"
	LB_STATUS_DISABLED = "disabled"

	LB_STATUS_INIT = "init"

	LB_CREATING = "creating"

	LB_SYNC_CONF = "sync_conf"

	LB_STATUS_DELETING = "deleting"
	LB_STATUS_DELETED  = "deleted"

	LB_STATUS_START_FAILED = "start_failed"

	LB_STATUS_UNKNOWN = "unknown"
)

const (
	//默认后端服务器组
	LB_BACKENDGROUP_TYPE_DEFAULT = "default"
	//普通后端服务器组
	LB_BACKENDGROUP_TYPE_NORMAL = "normal"
	//主备后端服务器组
	LB_BACKENDGROUP_TYPE_MASTER_SLAVE = "master_slave"
)

const (
	LB_AWS_SPEC_APPLICATION = "application"
	LB_AWS_SPEC_NETWORK     = "network"
)

// Load Balancer network type (vpc or classic) determines viable backend
// servers (they should be from the same network type as the load balancer).
//
// Load Balancer address type (intranet or internet) determins the scope the
// service provided by load balancer can be accessed.  If it's intranet, then
// it will only be accessible from the specified network.  If it's internet,
// then it's public and can be accessed from outside the cloud region
const (
	LB_ADDR_TYPE_INTRANET = "intranet"
	LB_ADDR_TYPE_INTERNET = "internet"
)

const (
	LB_NETWORK_TYPE_CLASSIC = "classic"
	LB_NETWORK_TYPE_VPC     = "vpc"
)

// TODO https_direct sni
const (
	LB_LISTENER_TYPE_TCP              = "tcp"
	LB_LISTENER_TYPE_UDP              = "udp"
	LB_LISTENER_TYPE_TCP_UDP          = "tcp_udp"
	LB_LISTENER_TYPE_HTTP             = "http"
	LB_LISTENER_TYPE_HTTPS            = "https"
	LB_LISTENER_TYPE_TERMINATED_HTTPS = "terminated_https"
)

const (
	LB_ACL_TYPE_BLACK = "black"
	LB_ACL_TYPE_WHITE = "white"
)

const (
	LB_TLS_CERT_FINGERPRINT_ALGO_SHA1   = "sha1"
	LB_TLS_CERT_FINGERPRINT_ALGO_SHA256 = "sha256"
)

const (
	LB_STICKY_SESSION_TYPE_INSERT = "insert"
	LB_STICKY_SESSION_TYPE_SERVER = "server"
)

// TODO maybe https check when field need comes ;)
const (
	LB_HEALTH_CHECK_PING  = "ping"
	LB_HEALTH_CHECK_TCP   = "tcp"
	LB_HEALTH_CHECK_UDP   = "udp"
	LB_HEALTH_CHECK_HTTP  = "http"
	LB_HEALTH_CHECK_HTTPS = "https"
)

const (
	LB_HEALTH_CHECK_HTTP_CODE_1xx     = "http_1xx"
	LB_HEALTH_CHECK_HTTP_CODE_2xx     = "http_2xx"
	LB_HEALTH_CHECK_HTTP_CODE_3xx     = "http_3xx"
	LB_HEALTH_CHECK_HTTP_CODE_4xx     = "http_4xx"
	LB_HEALTH_CHECK_HTTP_CODE_5xx     = "http_5xx"
	LB_HEALTH_CHECK_HTTP_CODE_DEFAULT = "http_2xx,http_3xx"
)

const (
	LB_REDIRECT_OFF = "off"
	LB_REDIRECT_RAW = "raw"
)

const (
	LB_REDIRECT_CODE_301 = int64(301) // Moved Permanently
	LB_REDIRECT_CODE_302 = int64(302) // Found
	LB_REDIRECT_CODE_307 = int64(307) // Temporary Redirect
)

const (
	LB_BOOL_ON  = "on"
	LB_BOOL_OFF = "off"
)

// TODO
//
// - qch, quic connection id
// - mh, maglev consistent hash
const (
	LB_SCHEDULER_RR  = "rr"  // round robin
	LB_SCHEDULER_WRR = "wrr" // weighted round robin
	LB_SCHEDULER_WLC = "wlc" // weighted least connection
	LB_SCHEDULER_SCH = "sch" // source-ip-based consistent hash
	LB_SCHEDULER_TCH = "tch" // 4-tuple-based consistent hash
	LB_SCHEDULER_QCH = "qch"
	LB_SCHEDULER_MH  = "mh" // maglev consistent hash
)

// TODO raw type
const (
	LB_BACKEND_GUEST = "guest"
	LB_BACKEND_HOST  = "host"
	LB_BACKEND_IP    = "ip"
)

const (
	LB_BACKEND_ROLE_DEFAULT = "default"
	LB_BACKEND_ROLE_MASTER  = "master"
	LB_BACKEND_ROLE_SLAVE   = "slave"
)

const (
	LB_CHARGE_TYPE_BY_TRAFFIC   = "traffic"
	LB_CHARGE_TYPE_BY_BANDWIDTH = "bandwidth"
)
