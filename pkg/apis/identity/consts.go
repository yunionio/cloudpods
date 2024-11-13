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

package identity

import "yunion.io/x/onecloud/pkg/apis"

const (
	SERVICE_TYPE = apis.SERVICE_TYPE_KEYSTONE

	DEFAULT_DOMAIN_ID   = "default"
	DEFAULT_DOMAIN_NAME = "Default"

	DefaultRemoteDomainId = "default_domain"

	DEFAULT_IDP_ID = DEFAULT_DOMAIN_ID

	SystemAdminUser    = "sysadmin"
	SystemAdminProject = "system"
	SystemAdminRole    = "admin"

	AUTH_METHOD_PASSWORD = "password"
	AUTH_METHOD_TOKEN    = "token"
	AUTH_METHOD_AKSK     = "aksk"
	AUTH_METHOD_CAS      = "cas"
	AUTH_METHOD_SAML     = "saml"
	AUTH_METHOD_OIDC     = "oidc"
	AUTH_METHOD_OAuth2   = "oauth2"
	AUTH_METHOD_VERIFY   = "verify"

	// AUTH_METHOD_ID_PASSWORD = 1
	// AUTH_METHOD_ID_TOKEN    = 2

	AUTH_TOKEN_HEADER         = "X-Auth-Token"
	AUTH_SUBJECT_TOKEN_HEADER = "X-Subject-Token"

	AssignmentUserProject  = "UserProject"
	AssignmentGroupProject = "GroupProject"
	AssignmentUserDomain   = "UserDomain"
	AssignmentGroupDomain  = "GroupDomain"

	EndpointInterfacePublic   = "public"
	EndpointInterfaceInternal = "internal"
	EndpointInterfaceAdmin    = "admin"
	EndpointInterfaceConsole  = "console"

	EndpointInterfaceApigateway = "apigateway"

	KeystoneDomainRoot = "<<keystone.domain.root>>"

	IdMappingEntityUser   = "user"
	IdMappingEntityGroup  = "group"
	IdMappingEntityDomain = "domain"

	IdentityDriverSQL    = "sql"
	IdentityDriverLDAP   = "ldap"
	IdentityDriverCAS    = "cas"
	IdentityDriverSAML   = "saml"
	IdentityDriverOIDC   = "oidc"   // OpenID Connect
	IdentityDriverOAuth2 = "oauth2" // OAuth2.0

	IdentityDriverStatusConnected    = "connected"
	IdentityDriverStatusDisconnected = "disconnected"
	IdentityDriverStatusDeleting     = "deleting"
	IdentityDriverStatusDeleteFailed = "delete_fail"

	IdentityProviderSyncLocal  = "local"
	IdentityProviderSyncFull   = "full"
	IdentityProviderSyncOnAuth = "auth"

	IdentitySyncStatusQueued  = "queued"
	IdentitySyncStatusSyncing = "syncing"
	IdentitySyncStatusIdle    = "idle"

	MinimalSyncIntervalSeconds = 5 * 60 // 5 minutes

	AUTH_TOKEN_LENGTH = 64
)

var (
	AUTH_METHODS = []string{AUTH_METHOD_PASSWORD, AUTH_METHOD_TOKEN, AUTH_METHOD_AKSK, AUTH_METHOD_CAS}

	PASSWORD_PROTECTED_IDPS = []string{
		IdentityDriverSQL,
		IdentityDriverLDAP,
	}

	SensitiveDomainConfigMap = map[string][]string{
		"ldap": {
			"password",
		},
	}

	CommonWhitelistOptionMap = map[string][]string{
		"default": {
			"enable_quota_check",
			"default_quota_value",
			"non_default_domain_projects",
			"time_zone",
			"domainized_namespace",
			"api_server",
			"customized_private_prefixes",
			"global_http_proxy",
			"global_https_proxy",
			"ignore_nonrunning_guests",
			"platform_name",
			"enable_cloud_shell",
			"platform_names",
			"enable_change_owner_auto_rename",
		},
	}

	ServiceBlacklistOptionMap = map[string][]string{
		"default": {
			// ############################
			// common blacklist options
			// ############################
			"help",
			"version",
			"config",
			"pid_file",

			"region",
			"application_id",
			"log_level",
			"log_verbose_level",
			"temp_path",
			"address",
			"port",
			"port_v2",
			"admin_port",
			"notify_admin_users",
			"session_endpoint_type",
			"admin_password",
			"admin_project",
			"admin_project_domain",
			"admin_user",
			"admin_domain",
			"auth_url",
			"enable_ssl",
			"ssl_certfile",
			"ssl_keyfile",
			"ssl_ca_certs",

			"is_slave_node",
			"config_sync_period_seconds",
			"enable_app_profiling",

			// ############################
			// db blacklist options
			// ############################
			"sql_connection",
			"clickhouse",
			"ops_log_with_clickhouse",
			"db_checksum_skip_init",
			"db_checksum_tables",
			"enable_db_checksum_tables",
			"db_checksum_hash_algorithm",
			"auto_sync_table",
			"exit_after_db_init",
			"global_virtual_resource_namespace",
			"debug_sqlchemy",
			"lockman_method",
			"etcd_lock_prefix",
			"etcd_lock_ttl",
			"etcd_endpoints",
			"etcd_username",
			"etcd_password",
			"etcd_use_tls",
			"etcd_skip_tls_verify",
			"etcd_cacert",
			"etcd_cert",
			"etcd_key",
			"splitable_max_duration_hours",
			"splitable_max_keep_segments",
			"ops_log_max_keep_months",

			"disable_local_vpc",

			// ############################
			// keystone blacklist options
			// ############################
			"bootstrap_admin_user_password",
			"reset_admin_user_password",
			"fernet_key_repository",

			// ############################
			// baremetal blacklist options
			// ############################
			"listen_interface",
			"access_address",
			"listen_address",
			"tftp_root",
			// "AutoRegisterBaremetal",
			"baremetals_path",
			// "LinuxDefaultRootUser",
			"ipmi_lan_port_shared",
			"zone",
			"dhcp_lease_time",
			"dhcp_renewal_time",
			"enable_general_guest_dhcp",
			"force_dhcp_probe_ipmi",
			"tftp_block_size_in_bytes",
			"tftp_max_timeout_retries",
			"enable_grub_tftp_download",
			"lengthy_worker_count",
			"short_worker_count",
			// "default_ipmi_password",
			// "default_strong_ipmi_password",
			// "windows_default_admin_user",
			"cache_path",
			"enable_pxe_boot",
			"boot_iso_path",
			// "status_probe_interval_seconds",
			// "log_fetch_interval_seconds",
			// "send_metrics_interval_seconds",
			// ############################
			// glance blacklist options
			// ############################
			"deploy_server_socket_path",
			"enable_remote_executor",
			"executor_socket_path",

			// ############################
			// kubeserver blacklist options
			// ############################
			"running_mode",
			"enable_default_policy",
		},
	}
)

func mergeConfigOptionsFrom(opt1, opt2 map[string][]string) map[string][]string {
	for opt, values := range opt2 {
		ovalues, _ := opt1[opt]
		opt1[opt] = append(ovalues, values...)
	}
	return opt1
}

func MergeServiceConfigOptions(opts ...map[string][]string) map[string][]string {
	ret := make(map[string][]string)
	for i := range opts {
		ret = mergeConfigOptionsFrom(ret, opts[i])
	}
	return ret
}
