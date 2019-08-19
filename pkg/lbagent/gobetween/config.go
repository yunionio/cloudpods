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

/**
 * mod_config.go - config file definitions
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 * @author Gene Ponomarenko <kikomdev@gmail.com>
 */
package gobetween

/**
 * Config file top-level object
 */
type Config struct {
	Logging  LoggingConfig     `toml:"logging" json:"logging"`
	Api      ApiConfig         `toml:"api" json:"api"`
	Defaults ConnectionOptions `toml:"defaults" json:"defaults"`
	Acme     *AcmeConfig       `toml:"acme" json:"acme"`
	Servers  map[string]Server `toml:"servers" json:"servers"`
}

/**
 * Logging config section
 */
type LoggingConfig struct {
	Level  string `toml:"level" json:"level"`
	Output string `toml:"output" json:"output"`
}

/**
 * Api config section
 */
type ApiConfig struct {
	Enabled   bool                `toml:"enabled" json:"enabled"`
	Bind      string              `toml:"bind" json:"bind"`
	BasicAuth *ApiBasicAuthConfig `toml:"basic_auth" json:"basic_auth"`
	Tls       *ApiTlsConfig       `toml:"tls" json:"tls"`
	Cors      bool                `toml:"cors" json:"cors"`
}

/**
 * Api Basic Auth Config
 */
type ApiBasicAuthConfig struct {
	Login    string `toml:"login" json:"login"`
	Password string `toml:"password" json:"password"`
}

/**
 * Api TLS server Config
 */
type ApiTlsConfig struct {
	CertPath string `toml:"cert_path" json:"cert_path"`
	KeyPath  string `toml:"key_path" json:"key_path"`
}

/**
 * Default values can be overridden in server
 */
type ConnectionOptions struct {
	MaxConnections           *int    `toml:"max_connections" json:"max_connections"`
	ClientIdleTimeout        *string `toml:"client_idle_timeout" json:"client_idle_timeout"`
	BackendIdleTimeout       *string `toml:"backend_idle_timeout" json:"backend_idle_timeout"`
	BackendConnectionTimeout *string `toml:"backend_connection_timeout" json:"backend_connection_timeout"`
}

/**
 * Acme config
 */
type AcmeConfig struct {
	Challenge string `toml:"challenge" json:"challenge"`
	HttpBind  string `toml:"http_bind" json:"http_bind"`
	CacheDir  string `toml:"cache_dir" json:"cache_dir"`
}

/**
 * Server section config
 */
type Server struct {
	ConnectionOptions

	// hostname:port
	Bind string `toml:"bind" json:"bind"`

	// tcp | udp | tls
	Protocol string `toml:"protocol" json:"protocol"`

	// weight | leastconn | roundrobin
	Balance string `toml:"balance" json:"balance"`

	// Optional configuration for server name indication
	Sni *Sni `toml:"sni" json:"sni"`

	// Optional configuration for protocol = tls
	Tls *Tls `toml:"tls" json:"tls"`

	// Optional configuration for backend_tls_enabled = true
	BackendsTls *BackendsTls `toml:"backends_tls" json:"backends_tls"`

	// Optional configuration for protocol = udp
	Udp *Udp `toml:"udp" json:"udp"`

	// Access configuration
	Access *AccessConfig `toml:"access" json:"access"`

	// ProxyProtocol configuration
	ProxyProtocol *ProxyProtocol `toml:"proxy_protocol" json:"proxy_protocol"`

	// Discovery configuration
	Discovery *DiscoveryConfig `toml:"discovery" json:"discovery"`

	// Healthcheck configuration
	Healthcheck *HealthcheckConfig `toml:"healthcheck" json:"healthcheck"`
}

/**
 * ProxyProtocol configurtion
 */
type ProxyProtocol struct {
	Version string `toml:"version" json:"version"`
}

/**
 * Server Sni options
 */
type Sni struct {
	HostnameMatchingStrategy   string `toml:"hostname_matching_strategy" json:"hostname_matching_strategy"`
	UnexpectedHostnameStrategy string `toml:"unexpected_hostname_strategy" json:"unexpected_hostname_strategy"`
	ReadTimeout                string `toml:"read_timeout" json:"read_timeout"`
}

/**
 * Common part of Tls and BackendTls types
 */
type tlsCommon struct {
	Ciphers             []string `toml:"ciphers" json:"ciphers"`
	PreferServerCiphers bool     `toml:"prefer_server_ciphers" json:"prefer_server_ciphers"`
	MinVersion          string   `toml:"min_version" json:"min_version"`
	MaxVersion          string   `toml:"max_version" json:"max_version"`
	SessionTickets      bool     `toml:"session_tickets" json:"session_tickets"`
}

/**
 * Server Tls options
 * for protocol = "tls"
 */
type Tls struct {
	AcmeHosts []string `toml:"acme_hosts" json:"acme_hosts"`
	CertPath  string   `toml:"cert_path" json:"cert_path"`
	KeyPath   string   `toml:"key_path" json:"key_path"`
	tlsCommon
}

type BackendsTls struct {
	IgnoreVerify   bool    `toml:"ignore_verify" json:"ignore_verify"`
	RootCaCertPath *string `toml:"root_ca_cert_path" json:"root_ca_cert_path"`
	CertPath       *string `toml:"cert_path" json:"cert_path"`
	KeyPath        *string `toml:"key_path" json:"key_path"`
	tlsCommon
}

/**
 * Server udp options
 * for protocol = "udp"
 */
type Udp struct {
	MaxRequests  uint64 `toml:"max_requests" json:"max_requests"`
	MaxResponses uint64 `toml:"max_responses" json:"max_responses"`
}

/**
 * Access configuration
 */
type AccessConfig struct {
	Default string   `toml:"default" json:"default"`
	Rules   []string `toml:"rules" json:"rules"`
}

/**
 * Discovery configuration
 */
type DiscoveryConfig struct {
	Kind       string `toml:"kind" json:"kind"`
	Failpolicy string `toml:"failpolicy" json:"failpolicy"`
	Interval   string `toml:"interval" json:"interval"`
	Timeout    string `toml:"timeout" json:"timeout"`

	/* Depends on Kind */

	*StaticDiscoveryConfig
	*SrvDiscoveryConfig
	*DockerDiscoveryConfig
	*JsonDiscoveryConfig
	*ExecDiscoveryConfig
	*PlaintextDiscoveryConfig
	*ConsulDiscoveryConfig
	*LXDDiscoveryConfig
}

type StaticDiscoveryConfig struct {
	StaticList []string `toml:"static_list" json:"static_list"`
}

type SrvDiscoveryConfig struct {
	SrvLookupServer  string `toml:"srv_lookup_server" json:"srv_lookup_server"`
	SrvLookupPattern string `toml:"srv_lookup_pattern" json:"srv_lookup_pattern"`
	SrvDnsProtocol   string `toml:"srv_dns_protocol" json:"srv_dns_protocol"`
}

type ExecDiscoveryConfig struct {
	ExecCommand []string `toml:"exec_command" json:"exec_command"`
}

type JsonDiscoveryConfig struct {
	JsonEndpoint        string `toml:"json_endpoint" json:"json_endpoint"`
	JsonHostPattern     string `toml:"json_host_pattern" json:"json_host_pattern"`
	JsonPortPattern     string `toml:"json_port_pattern" json:"json_port_pattern"`
	JsonWeightPattern   string `toml:"json_weight_pattern" json:"json_weight_pattern"`
	JsonPriorityPattern string `toml:"json_priority_pattern" json:"json_priority_pattern"`
	JsonSniPattern      string `toml:"json_sni_pattern" json:"json_sni_pattern"`
}

type PlaintextDiscoveryConfig struct {
	PlaintextEndpoint      string `toml:"plaintext_endpoint" json:"plaintext_endpoint"`
	PlaintextRegexpPattern string `toml:"plaintext_regex_pattern" json:"plaintext_regex_pattern"`
}

type DockerDiscoveryConfig struct {
	DockerEndpoint             string `toml:"docker_endpoint" json:"docker_endpoint"`
	DockerContainerLabel       string `toml:"docker_container_label" json:"docker_container_label"`
	DockerContainerPrivatePort int64  `toml:"docker_container_private_port" json:"docker_container_private_port"`
	DockerContainerHostEnvVar  string `toml:"docker_container_host_env_var" json:"docker_container_host_env_var"`

	DockerTlsEnabled    bool   `toml:"docker_tls_enabled" json:"docker_tls_enabled"`
	DockerTlsCertPath   string `toml:"docker_tls_cert_path" json:"docker_tls_cert_path"`
	DockerTlsKeyPath    string `toml:"docker_tls_key_path" json:"docker_tls_key_path"`
	DockerTlsCacertPath string `toml:"docker_tls_cacert_path" json:"docker_tls_cacert_path"`
}

type ConsulDiscoveryConfig struct {
	ConsulHost               string `toml:"consul_host" json:"consul_host"`
	ConsulServiceName        string `toml:"consul_service_name" json:"consul_service_name"`
	ConsulServiceTag         string `toml:"consul_service_tag" json:"consul_service_tag"`
	ConsulServicePassingOnly bool   `toml:"consul_service_passing_only" json:"consul_service_passing_only"`
	ConsulDatacenter         string `toml:"consul_datacenter" json:"consul_datacenter"`

	ConsulAuthUsername string `toml:"consul_auth_username" json:"consul_auth_username"`
	ConsulAuthPassword string `toml:"consul_auth_password" json:"consul_auth_password"`

	ConsulTlsEnabled    bool   `toml:"consul_tls_enabled" json:"consul_tls_enabled"`
	ConsulTlsCertPath   string `toml:"consul_tls_cert_path" json:"consul_tls_cert_path"`
	ConsulTlsKeyPath    string `toml:"consul_tls_key_path" json:"consul_tls_key_path"`
	ConsulTlsCacertPath string `toml:"consul_tls_cacert_path" json:"consul_tls_cacert_path"`
}

type LXDDiscoveryConfig struct {
	LXDServerAddress        string `toml:"lxd_server_address" json:"lxd_server_address"`
	LXDServerRemoteName     string `toml:"lxd_server_remote_name" json:"lxd_server_remote_name"`
	LXDServerRemotePassword string `toml:"lxd_server_remote_password" json:"lxd_server_remote_password"`

	LXDConfigDirectory     string `toml:"lxd_config_directory" json:"lxd_config_directory"`
	LXDGenerateClientCerts bool   `toml:"lxd_generate_client_certs" json:"lxd_generate_client_certs"`
	LXDAcceptServerCert    bool   `toml:"lxd_accept_server_cert" json:"lxd_accept_server_cert"`

	LXDContainerLabelKey   string `toml:"lxd_container_label_key" json:"lxd_container_label_key"`
	LXDContainerLabelValue string `toml:"lxd_container_label_value" json:"lxd_container_label_value"`

	LXDContainerPort    int    `toml:"lxd_container_port" json:"lxd_container_port"`
	LXDContainerPortKey string `toml:"lxd_container_port_key" json:"lxd_container_port_key"`

	LXDContainerInterface    string `toml:"lxd_container_interface" json:"lxd_container_interface"`
	LXDContainerInterfaceKey string `toml:"lxd_container_interface_key" json:"lxd_container_interface_key"`

	LXDContainerSNIKey      string `toml:"lxd_container_sni_key" json:"lxd_container_sni_key"`
	LXDContainerAddressType string `toml:"lxd_container_address_type" json:"lxd_container_address_type"`
}

/**
 * Healthcheck configuration
 */
type HealthcheckConfig struct {
	Kind     string `toml:"kind" json:"kind"`
	Interval string `toml:"interval" json:"interval"`
	Passes   int    `toml:"passes" json:"passes"`
	Fails    int    `toml:"fails" json:"fails"`
	Timeout  string `toml:"timeout" json:"timeout"`

	/* Depends on Kind */

	*PingHealthcheckConfig
	*ExecHealthcheckConfig
	*UdpHealthcheckConfig
}

type PingHealthcheckConfig struct{}

type ExecHealthcheckConfig struct {
	ExecCommand                string `toml:"exec_command" json:"exec_command,omitempty"`
	ExecExpectedPositiveOutput string `toml:"exec_expected_positive_output" json:"exec_expected_positive_output"`
	ExecExpectedNegativeOutput string `toml:"exec_expected_negative_output" json:"exec_expected_negative_output"`
}

type UdpHealthcheckConfig struct {
	Receive string
	Send    string
}
