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
	"yunion.io/x/onecloud/pkg/util/choices"
)

// Load balancer status transition (for spec status)
//
//                      create          start           stop            delete
//      init            running         -               -               -
//      running		-		-		stopped		stopped
//      stopped		-		running		-		-
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
//
const (
	LB_STATUS_ENABLED  = "enabled"
	LB_STATUS_DISABLED = "disabled"

	LB_STATUS_INIT = "init"

	LB_CREATING      = "creating"
	LB_CREATE_FAILED = "create_failed"

	LB_SYNC_CONF        = "sync_conf"
	LB_SYNC_CONF_FAILED = "sync_conf_failed"

	LB_STATUS_DELETING      = "deleting"
	LB_STATUS_DELETE_FAILED = "delete_failed"
	LB_STATUS_DELETED       = "deleted"

	LB_STATUS_START_FAILED = "start_failed"
	LB_STATUS_STOP_FAILED  = "stop_failed"

	LB_UPDATE_TAGS        = "update_tags"
	LB_UPDATE_TAGS_FAILED = "update_tags_fail"

	LB_STATUS_UNKNOWN = "unknown"
)

var LB_STATUS_SPEC = choices.NewChoices(
	LB_STATUS_ENABLED,
	LB_STATUS_DISABLED,
)

const (
	//默认后端服务器组
	LB_BACKENDGROUP_TYPE_DEFAULT = "default"
	//普通后端服务器组
	LB_BACKENDGROUP_TYPE_NORMAL = "normal"
	//主备后端服务器组
	LB_BACKENDGROUP_TYPE_MASTER_SLAVE = "master_slave"
)

var LB_BACKENDGROUP_TYPE = choices.NewChoices(
	LB_BACKENDGROUP_TYPE_DEFAULT,
	LB_BACKENDGROUP_TYPE_NORMAL,
	LB_BACKENDGROUP_TYPE_MASTER_SLAVE,
)

const (
	LB_ALIYUN_SPEC_SHAREABLE = "" //性能共享型
	LB_ALIYUN_SPEC_S1_SMALL  = "slb.s1.small"
	LB_ALIYUN_SPEC_S2_SMALL  = "slb.s2.small"
	LB_ALIYUN_SPEC_S3_SMALL  = "slb.s3.small"
	LB_ALIYUN_SPEC_S2_MEDIUM = "slb.s2.medium"
	LB_ALIYUN_SPEC_S3_MEDIUM = "slb.s3.medium"
	LB_ALIYUN_SPEC_S3_LARGE  = "slb.s3.large"

	LB_AWS_SPEC_APPLICATION = "application"
	LB_AWS_SPEC_NETWORK     = "network"
)

const (
	LB_MbpsMin = 0
	LB_MbpsMax = 10000
)

var LB_ALIYUN_SPECS = choices.NewChoices(
	LB_ALIYUN_SPEC_SHAREABLE,
	LB_ALIYUN_SPEC_S1_SMALL,
	LB_ALIYUN_SPEC_S2_SMALL,
	LB_ALIYUN_SPEC_S3_SMALL,
	LB_ALIYUN_SPEC_S2_MEDIUM,
	LB_ALIYUN_SPEC_S3_MEDIUM,
	LB_ALIYUN_SPEC_S3_LARGE,
)

var LB_AWS_SPECS = choices.NewChoices(
	LB_AWS_SPEC_APPLICATION,
	LB_AWS_SPEC_NETWORK,
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

var LB_ADDR_TYPES = choices.NewChoices(
	LB_ADDR_TYPE_INTERNET,
	LB_ADDR_TYPE_INTRANET,
)

const (
	LB_NETWORK_TYPE_CLASSIC = "classic"
	LB_NETWORK_TYPE_VPC     = "vpc"
)

var LB_NETWORK_TYPES = choices.NewChoices(
	LB_NETWORK_TYPE_CLASSIC,
	LB_NETWORK_TYPE_VPC,
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

var LB_LISTENER_TYPES = choices.NewChoices(
	LB_LISTENER_TYPE_TCP,
	LB_LISTENER_TYPE_UDP,
	LB_LISTENER_TYPE_HTTP,
	LB_LISTENER_TYPE_HTTPS,
)

// aws_network_lb_listener
var AWS_NETWORK_LB_LISTENER_TYPES = choices.NewChoices(
	LB_LISTENER_TYPE_TCP,
	LB_LISTENER_TYPE_UDP,
	// LB_LISTENER_TYPE_TCP_UDP
)

// aws_application_lb_listener
var AWS_APPLICATION_LB_LISTENER_TYPES = choices.NewChoices(
	LB_LISTENER_TYPE_HTTP,
	LB_LISTENER_TYPE_HTTPS,
)

// huawei backend group protocal choices
var HUAWEI_LBBG_PROTOCOL_TYPES = choices.NewChoices(
	LB_LISTENER_TYPE_TCP,
	LB_LISTENER_TYPE_UDP,
	LB_LISTENER_TYPE_HTTP,
)

var HUAWEI_LBBG_SCHDULERS = choices.NewChoices(
	LB_SCHEDULER_WLC,
	LB_SCHEDULER_RR,
	LB_SCHEDULER_SCH,
)

const (
	LB_ACL_TYPE_BLACK = "black"
	LB_ACL_TYPE_WHITE = "white"
)

var LB_ACL_TYPES = choices.NewChoices(
	LB_ACL_TYPE_BLACK,
	LB_ACL_TYPE_WHITE,
)

const (
	LB_TLS_CERT_FINGERPRINT_ALGO_SHA1   = "sha1"
	LB_TLS_CERT_FINGERPRINT_ALGO_SHA256 = "sha256"
)

const (
	LB_TLS_CERT_PUBKEY_ALGO_RSA   = "RSA"
	LB_TLS_CERT_PUBKEY_ALGO_ECDSA = "ECDSA"
)

var LB_TLS_CERT_PUBKEY_ALGOS = choices.NewChoices(
	LB_TLS_CERT_PUBKEY_ALGO_RSA,
	LB_TLS_CERT_PUBKEY_ALGO_ECDSA,
)

// TODO may want extra for legacy apps
const (
	LB_TLS_CIPHER_POLICY_1_0        = "tls_cipher_policy_1_0"
	LB_TLS_CIPHER_POLICY_1_1        = "tls_cipher_policy_1_1"
	LB_TLS_CIPHER_POLICY_1_2        = "tls_cipher_policy_1_2"
	LB_TLS_CIPHER_POLICY_1_2_strict = "tls_cipher_policy_1_2_strict"
	LB_TLS_CIPHER_POLICY_deault     = ""
)

var LB_TLS_CIPHER_POLICIES = choices.NewChoices(
	LB_TLS_CIPHER_POLICY_1_0,
	LB_TLS_CIPHER_POLICY_1_1,
	LB_TLS_CIPHER_POLICY_1_2,
	LB_TLS_CIPHER_POLICY_1_2_strict,
	LB_TLS_CIPHER_POLICY_deault,
)

const (
	LB_STICKY_SESSION_TYPE_INSERT = "insert"
	LB_STICKY_SESSION_TYPE_SERVER = "server"
)

var LB_STICKY_SESSION_TYPES = choices.NewChoices(
	LB_STICKY_SESSION_TYPE_INSERT,
	LB_STICKY_SESSION_TYPE_SERVER,
)

// TODO maybe https check when field need comes ;)
const (
	LB_HEALTH_CHECK_PING  = "ping"
	LB_HEALTH_CHECK_TCP   = "tcp"
	LB_HEALTH_CHECK_UDP   = "udp"
	LB_HEALTH_CHECK_HTTP  = "http"
	LB_HEALTH_CHECK_HTTPS = "https"
)

var LB_HEALTH_CHECK_TYPES = choices.NewChoices(
	LB_HEALTH_CHECK_TCP,
	LB_HEALTH_CHECK_UDP,
	LB_HEALTH_CHECK_HTTP,
)

var LB_HEALTH_CHECK_TYPES_TCP = choices.NewChoices(
	LB_HEALTH_CHECK_TCP,
	LB_HEALTH_CHECK_HTTP,
)

var LB_HEALTH_CHECK_TYPES_UDP = choices.NewChoices(
	LB_HEALTH_CHECK_UDP,
)

const (
	LB_HEALTH_CHECK_HTTP_CODE_1xx     = "http_1xx"
	LB_HEALTH_CHECK_HTTP_CODE_2xx     = "http_2xx"
	LB_HEALTH_CHECK_HTTP_CODE_3xx     = "http_3xx"
	LB_HEALTH_CHECK_HTTP_CODE_4xx     = "http_4xx"
	LB_HEALTH_CHECK_HTTP_CODE_5xx     = "http_5xx"
	LB_HEALTH_CHECK_HTTP_CODE_DEFAULT = "http_2xx,http_3xx"
)

var LB_HEALTH_CHECK_HTTP_CODES = choices.NewChoices(
	LB_HEALTH_CHECK_HTTP_CODE_1xx,
	LB_HEALTH_CHECK_HTTP_CODE_2xx,
	LB_HEALTH_CHECK_HTTP_CODE_3xx,
	LB_HEALTH_CHECK_HTTP_CODE_4xx,
	LB_HEALTH_CHECK_HTTP_CODE_5xx,
)

const (
	LB_REDIRECT_OFF = "off"
	LB_REDIRECT_RAW = "raw"
)

var LB_REDIRECT_TYPES = choices.NewChoices(
	LB_REDIRECT_OFF,
	LB_REDIRECT_RAW,
)

const (
	LB_REDIRECT_CODE_301 = int64(301) // Moved Permanently
	LB_REDIRECT_CODE_302 = int64(302) // Found
	LB_REDIRECT_CODE_307 = int64(307) // Temporary Redirect
)

var LB_REDIRECT_CODES = []int64{
	LB_REDIRECT_CODE_301,
	LB_REDIRECT_CODE_302,
	LB_REDIRECT_CODE_307,
}

const (
	LB_REDIRECT_SCHEME_IDENTITY = ""
	LB_REDIRECT_SCHEME_HTTP     = "http"
	LB_REDIRECT_SCHEME_HTTPS    = "https"
)

var LB_REDIRECT_SCHEMES = choices.NewChoices(
	LB_REDIRECT_SCHEME_IDENTITY,
	LB_REDIRECT_SCHEME_HTTP,
	LB_REDIRECT_SCHEME_HTTPS,
)

const (
	LB_BOOL_ON  = "on"
	LB_BOOL_OFF = "off"
)

var LB_BOOL_VALUES = choices.NewChoices(
	LB_BOOL_ON,
	LB_BOOL_OFF,
)

//TODO
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

var LB_SCHEDULER_TYPES = choices.NewChoices(
	LB_SCHEDULER_RR,
	LB_SCHEDULER_WRR,
	LB_SCHEDULER_WLC,
	LB_SCHEDULER_SCH,
	LB_SCHEDULER_TCH,
)

const (
	LB_SENDPROXY_OFF       = "off"
	LB_SENDPROXY_V1        = "v1"
	LB_SENDPROXY_V2        = "v2"
	LB_SENDPROXY_V2_SSL    = "v2-ssl"
	LB_SENDPROXY_V2_SSL_CN = "v2-ssl-cn"
)

var LB_SENDPROXY_CHOICES = choices.NewChoices(
	LB_SENDPROXY_OFF,
	LB_SENDPROXY_V1,
	LB_SENDPROXY_V2,
	LB_SENDPROXY_V2_SSL,
	LB_SENDPROXY_V2_SSL_CN,
)

var LB_ALIYUN_UDP_SCHEDULER_TYPES = choices.NewChoices(
	LB_SCHEDULER_RR,
	LB_SCHEDULER_WRR,
	LB_SCHEDULER_WLC,
	LB_SCHEDULER_SCH,
	LB_SCHEDULER_TCH,
	LB_SCHEDULER_QCH,
)

var LB_ALIYUN_COMMON_SCHEDULER_TYPES = choices.NewChoices(
	LB_SCHEDULER_RR,
	LB_SCHEDULER_WRR,
	LB_SCHEDULER_WLC,
)

// TODO raw type
const (
	LB_BACKEND_GUEST = "guest"
	LB_BACKEND_HOST  = "host"
	LB_BACKEND_IP    = "ip"
)

var LB_BACKEND_TYPES = choices.NewChoices(
	LB_BACKEND_GUEST,
	LB_BACKEND_HOST,
	LB_BACKEND_IP,
)

const (
	LB_BACKEND_ROLE_DEFAULT = "default"
	LB_BACKEND_ROLE_MASTER  = "master"
	LB_BACKEND_ROLE_SLAVE   = "slave"
)

var LB_BACKEND_ROLES = choices.NewChoices(
	LB_BACKEND_ROLE_MASTER,
	LB_BACKEND_ROLE_DEFAULT,
	LB_BACKEND_ROLE_SLAVE,
)

const (
	LB_CHARGE_TYPE_BY_TRAFFIC   = "traffic"
	LB_CHARGE_TYPE_BY_BANDWIDTH = "bandwidth"
	LB_CHARGE_TYPE_POSTPAID     = "postpaid"
)

var LB_CHARGE_TYPES = choices.NewChoices(
	LB_CHARGE_TYPE_BY_TRAFFIC,
	LB_CHARGE_TYPE_BY_BANDWIDTH,
	LB_CHARGE_TYPE_POSTPAID,
)

const (
	LB_HA_STATE_MASTER  = "MASTER"
	LB_HA_STATE_BACKUP  = "BACKUP"
	LB_HA_STATE_FAULT   = "FAULT"
	LB_HA_STATE_STOP    = "STOP"
	LB_HA_STATE_UNKNOWN = "UNKNOWN"
)

var LB_HA_STATES = choices.NewChoices(
	LB_HA_STATE_MASTER,
	LB_HA_STATE_BACKUP,
	LB_HA_STATE_FAULT,
	LB_HA_STATE_STOP,
	LB_HA_STATE_UNKNOWN,
)

const (
	LBAGENT_QUERY_ORIG_KEY = "_orig"
	LBAGENT_QUERY_ORIG_VAL = "lbagent"
)

const (
	LB_ASSOCIATE_TYPE_LISTENER = "listener"
	LB_ASSOCIATE_TYPE_RULE     = "rule"
)
