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

package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SKeystoneOptions struct {
	options.BaseOptions

	options.DBOptions

	AdminPort int `default:"35357" help:"listening port for admin API(deprecated)"`

	TokenExpirationSeconds int    `default:"86400" help:"token expiration seconds" token:"expiration"`
	FernetKeyRepository    string `help:"fernet key repo directory" token:"key_repository" default:"/etc/yunion/keystone/fernet-keys"`
	SetupCredentialKeys    bool   `help:"setup standalone fernet keys for credentials" token:"setup_credential_key" default:"false" json:",allowfalse"`

	BootstrapAdminUserPassword string `help:"bootstreap sysadmin user password" default:"sysadmin"`
	ResetAdminUserPassword     bool   `help:"reset sysadmin password if exists and this option is true" json:",allowfalse"`

	AutoSyncIntervalSeconds int `help:"frequency to check auto sync tasks" default:"30"`

	DefaultSyncIntervalSeconds int `help:"frequency to do auto sync tasks" default:"900"`

	FetchScopeResourceCountIntervalSeconds int `help:"frequency tp fetch project resource counts" default:"900"`

	PasswordExpirationSeconds  int `help:"password expires after the duration in seconds"`
	PasswordMinimalLength      int `help:"password minimal length" default:"6"`
	PasswordUniqueHistoryCheck int `help:"password must be unique in last N passwords"`
	PasswordCharComplexity     int `help:"password complexity policy" default:"0"`

	PasswordErrorLockCount int `help:"lock user account if given number of failed auth"`

	DefaultUserQuota    int `default:"500" help:"default quota for user per domain, default is 500"`
	DefaultGroupQuota   int `default:"500" help:"default quota for group per domain, default is 500"`
	DefaultProjectQuota int `default:"500" help:"default quota for project per domain, default is 500"`
	DefaultRoleQuota    int `default:"500" help:"default quota for role per domain, default is 500"`
	DefaultPolicyQuota  int `default:"500" help:"default quota for policy per domain, default is 500"`

	SessionEndpointType string `help:"Client session end point type"`

	// AllowJoinProjectsAcrossDomains bool `help:"allow users/groups to join projects across domains" default:"false"`

	DefaultUserLanguage string `help:"default user language, default to zh-CN" default:"zh-CN"`

	DomainAdminRoleToNotify string `help:"domain admin role to notify" default:"domainadmin"`
	AdminRoleToNotify       string `help:"admin role to notify" default:"admin"`

	EnableDefaultDashboardPolicy bool   `default:"true" help:"enable default dashboard policy"`
	SystemDashboardPolicy        string `help:"dashboard policy name for system view" default:""`
	DomainDashboardPolicy        string `help:"dashboard policy name for domain view" default:""`
	ProjectDashboardPolicy       string `help:"dashboard policy name for project view" default:""`

	NoPolicyViolationCheck bool `help:"do not check policy violation when modify or assign policy" default:"false"`

	ThreeAdminRoleSystem      bool     `help:"do not check policy violation when modify or assign policy" default:"false"`
	SystemThreeAdminRoleNames []string `help:"Name of system three-admin roles" default:"sys_secadmin,sys_opsadmin,sys_adtadmin"`
	DomainThreeAdminRoleNames []string `help:"Name of system three-admin roles" default:"domain_secadmin,domain_opsadmin,domain_adtadmin"`

	LdapSearchPageSize    uint32 `help:"pagination size for LDAP search" default:"100"`
	LdapSyncDisabledUsers bool   `help:"auto sync ldap disabled users"`

	ProjectAdminRole     string `help:"name of role to be saved as admin user of project" default:"project_owner"`
	PwdExpiredNotifyDays []int  `help:"The notify for password will expire " default:"1,7"`

	MaxUserRolesInProject  int `help:"maximal allowed roles of a user in a project" default:"20"`
	MaxGroupRolesInProject int `help:"maximal allowed roles of a group in a project" default:"20"`

	ForceEnableMfa string `help:"force enable mfa" default:"disable" choices:"all|after|disable"`
}

var (
	Options SKeystoneOptions
)

func OnOptionsChange(oldOptions, newOptions interface{}) bool {
	oldOpts := oldOptions.(*SKeystoneOptions)
	newOpts := newOptions.(*SKeystoneOptions)

	changed := false
	if options.OnBaseOptionsChange(&oldOpts.BaseOptions, &newOpts.BaseOptions) {
		changed = true
	}

	if options.OnDBOptionsChange(&oldOpts.DBOptions, &newOpts.DBOptions) {
		changed = true
	}

	if oldOpts.DefaultUserLanguage != newOpts.DefaultUserLanguage {
		changed = true
	}

	if oldOpts.ForceEnableMfa != newOpts.ForceEnableMfa {
		changed = true
	}

	return changed
}
