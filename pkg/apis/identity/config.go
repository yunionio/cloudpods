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

const (
	QueryScopeOne = "one"
	QUeryScopeSub = "sub"
)

type SDomainLDAPConfigOptions struct {
	Url        string `json:"url,omitempty" help:"LDAP server URL" required:"true"`
	Suffix     string `json:"suffix,omitempty" required:"true"`
	QueryScope string `json:"query_scope,omitempty" help:"Query scope, either one or sub" choices:"one|sub" default:"sub"`
	PageSize   int    `json:"page_size,omitzero" help:"Page size, default 20" default:"20"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`

	UserTreeDN              string   `json:"user_tree_dn,omitempty" help:"User tree distinguished name"`
	UserFilter              string   `json:"user_filter,omitempty"`
	UserObjectclass         string   `json:"user_objectclass,omitempty"`
	UserIdAttribute         string   `json:"user_id_attribute,omitempty"`
	UserNameAttribute       string   `json:"user_name_attribute,omitempty"`
	UserEnabledAttribute    string   `json:"user_enabled_attribute,omitempty"`
	UserEnabledMask         int64    `json:"user_enabled_mask,allowzero" default:"-1"`
	UserEnabledDefault      string   `json:"user_enabled_default,omitempty"`
	UserEnabledInvert       bool     `json:"user_enabled_invert,allowfalse"`
	UserAdditionalAttribute []string `json:"user_additional_attribute_mapping,omitempty" token:"user_additional_attribute"`

	GroupTreeDN          string `json:"group_tree_dn,omitempty" help:"Group tree distinguished name"`
	GroupFilter          string `json:"group_filter,omitempty"`
	GroupObjectclass     string `json:"group_objectclass,omitempty"`
	GroupIdAttribute     string `json:"group_id_attribute,omitempty"`
	GroupNameAttribute   string `json:"group_name_attribute,omitempty"`
	GroupMemberAttribute string `json:"group_member_attribute,omitempty"`
	GroupMembersAreIds   bool   `json:"group_members_are_ids,allowfalse"`
}
