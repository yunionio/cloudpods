package models

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
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

	LB_STATUS_INIT           = "init"
	LB_STATUS_RUNNING        = "running"
	LB_STATUS_STOPPED        = "stopped"
	LB_STATUS_CONFIGURING    = "configuring" // config changes pending
	LB_STATUS_STOPPING       = "stopping"
	LB_STATUS_DELETE_PENDING = "delete_pending"
	LB_STATUS_ERROR          = "error" // bad things happen
)

var LB_STATUS_SPEC = validators.NewChoices(
	LB_STATUS_ENABLED,
	LB_STATUS_DISABLED,
)

var LB_STATUS_RUNTIME = validators.NewChoices(
	LB_STATUS_INIT,
	LB_STATUS_CONFIGURING,
	LB_STATUS_RUNNING,
	LB_STATUS_STOPPING,
	LB_STATUS_STOPPED,
	LB_STATUS_ERROR,
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

var LB_ADDR_TYPES = validators.NewChoices(
	LB_ADDR_TYPE_INTERNET,
	LB_ADDR_TYPE_INTRANET,
)

const (
	LB_NETWORK_TYPE_CLASSIC = "classic"
	LB_NETWORK_TYPE_VPC     = "vpc"
)

var LB_NETWORK_TYPES = validators.NewChoices(
	LB_NETWORK_TYPE_CLASSIC,
	LB_NETWORK_TYPE_VPC,
)

// TODO https_direct sni
const (
	LB_LISTENER_TYPE_TCP   = "tcp"
	LB_LISTENER_TYPE_UDP   = "udp"
	LB_LISTENER_TYPE_HTTP  = "http"
	LB_LISTENER_TYPE_HTTPS = "https"
)

var LB_LISTENER_TYPES = validators.NewChoices(
	LB_LISTENER_TYPE_TCP,
	LB_LISTENER_TYPE_UDP,
	LB_LISTENER_TYPE_HTTP,
	LB_LISTENER_TYPE_HTTPS,
)

const (
	LB_ACL_TYPE_BLACK = "black"
	LB_ACL_TYPE_WHITE = "white"
)

var LB_ACL_TYPES = validators.NewChoices(
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

var LB_TLS_CERT_PUBKEY_ALGOS = validators.NewChoices(
	LB_TLS_CERT_PUBKEY_ALGO_RSA,
	LB_TLS_CERT_PUBKEY_ALGO_ECDSA,
)

// TODO may want extra for legacy apps
const (
	LB_TLS_CIPHER_POLICY_1_0        = "tls_cipher_policy_1_0"
	LB_TLS_CIPHER_POLICY_1_1        = "tls_cipher_policy_1_1"
	LB_TLS_CIPHER_POLICY_1_2        = "tls_cipher_policy_1_2"
	LB_TLS_CIPHER_POLICY_1_2_strict = "tls_cipher_policy_1_2_strict"
)

var LB_TLS_CIPHER_POLICIES = validators.NewChoices(
	LB_TLS_CIPHER_POLICY_1_0,
	LB_TLS_CIPHER_POLICY_1_1,
	LB_TLS_CIPHER_POLICY_1_2,
	LB_TLS_CIPHER_POLICY_1_2_strict,
)

const (
	LB_STICKY_SESSION_TYPE_INSERT = "insert"
	LB_STICKY_SESSION_TYPE_SERVER = "server"
)

var LB_STICKY_SESSION_TYPES = validators.NewChoices(
	LB_STICKY_SESSION_TYPE_INSERT,
	LB_STICKY_SESSION_TYPE_SERVER,
)

// TODO maybe https check when field need comes ;)
const (
	LB_HEALTH_CHECK_TCP  = "tcp"
	LB_HEALTH_CHECK_UDP  = "udp"
	LB_HEALTH_CHECK_HTTP = "http"
)

var LB_HEALTH_CHECK_TYPES = validators.NewChoices(
	LB_HEALTH_CHECK_TCP,
	LB_HEALTH_CHECK_UDP,
	LB_HEALTH_CHECK_HTTP,
)

var LB_HEALTH_CHECK_TYPES_TCP = validators.NewChoices(
	LB_HEALTH_CHECK_TCP,
	LB_HEALTH_CHECK_HTTP,
)

var LB_HEALTH_CHECK_TYPES_UDP = validators.NewChoices(
	LB_HEALTH_CHECK_UDP,
)

const (
	LB_HEALTH_CHECK_HTTP_CODE_2xx     = "http_2xx"
	LB_HEALTH_CHECK_HTTP_CODE_3xx     = "http_3xx"
	LB_HEALTH_CHECK_HTTP_CODE_4xx     = "http_4xx"
	LB_HEALTH_CHECK_HTTP_CODE_5xx     = "http_5xx"
	LB_HEALTH_CHECK_HTTP_CODE_DEFAULT = "http_2xx,http_3xx"
)

var LB_HEALTH_CHECK_HTTP_CODES = validators.NewChoices(
	LB_HEALTH_CHECK_HTTP_CODE_2xx,
	LB_HEALTH_CHECK_HTTP_CODE_3xx,
	LB_HEALTH_CHECK_HTTP_CODE_4xx,
	LB_HEALTH_CHECK_HTTP_CODE_5xx,
)

const (
	LB_BOOL_ON  = "on"
	LB_BOOL_OFF = "off"
)

var LB_BOOL_VALUES = validators.NewChoices(
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
)

var LB_SCHEDULER_TYPES = validators.NewChoices(
	LB_SCHEDULER_RR,
	LB_SCHEDULER_WRR,
	LB_SCHEDULER_WLC,
	LB_SCHEDULER_SCH,
	LB_SCHEDULER_TCH,
)

// TODO raw type
const (
	LB_BACKEND_GUEST = "guest"
	LB_BACKEND_HOST  = "host"
)

var LB_BACKEND_TYPES = validators.NewChoices(
	LB_BACKEND_GUEST,
	LB_BACKEND_HOST,
)
