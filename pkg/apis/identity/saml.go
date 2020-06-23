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

type SIdpAttributeOptions struct {
	UserNameAttribute string `json:"user_name_attribute"`
	UserIdAttribute   string `json:"user_id_attribute"`

	UserDisplaynameAttribtue string `json:"user_displayname_attribute"`
	UserEmailAttribute       string `json:"user_email_attribute"`
	UserMobileAttribute      string `json:"user_mobile_attribute"`

	ProjectAttribute string `json:"project_attribute"`
	RolesAttribute   string `json:"roles_attribute"`

	DefaultProjectId string `json:"default_project_id"`
	DefaultRoleId    string `json:"default_role_id"`
}

type SSAMLIdpConfigOptions struct {
	EntityId       string `json:"entity_id"`
	RedirectSSOUrl string `json:"redirect_sso_url"`

	SIdpAttributeOptions
}

type SSAMLTestIdpConfigOptions struct {
	// empty
}

type SSAMLAzureADConfigOptions struct {
	TenantId string `json:"tenant_id"`
}
