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

package ldap

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
)

var (
	MicrosoftActiveDirectorySingleDomainTemplate = api.SLDAPIdpConfigOptions{
		UserObjectclass:      "organizationalPerson",
		UserIdAttribute:      "sAMAccountName",
		UserNameAttribute:    "sAMAccountName",
		UserEnabledAttribute: "userAccountControl",
		UserEnabledMask:      2,
		UserEnabledDefault:   "512",
		UserEnabledInvert:    true,
		UserAdditionalAttribute: []string{
			"displayName:displayname",
			"telephoneNumber:mobile",
			"mail:email",
		},
		UserQueryScope:       "sub",
		GroupObjectclass:     "group",
		GroupIdAttribute:     "sAMAccountName",
		GroupNameAttribute:   "name",
		GroupMemberAttribute: "member",
		GroupMembersAreIds:   false,
		GroupQueryScope:      "sub",
	}

	MicrosoftActiveDirectoryMultipleDomainTemplate = api.SLDAPIdpConfigOptions{
		DomainObjectclass:    "organizationalUnit",
		DomainIdAttribute:    "objectGUID",
		DomainNameAttribute:  "name",
		DomainQueryScope:     "one",
		UserObjectclass:      "organizationalPerson",
		UserIdAttribute:      "sAMAccountName",
		UserNameAttribute:    "sAMAccountName",
		UserEnabledAttribute: "userAccountControl",
		UserEnabledMask:      2,
		UserEnabledDefault:   "512",
		UserEnabledInvert:    true,
		UserAdditionalAttribute: []string{
			"displayName:displayname",
			"telephoneNumber:mobile",
			"mail:email",
		},
		UserQueryScope:       "sub",
		GroupObjectclass:     "group",
		GroupIdAttribute:     "sAMAccountName",
		GroupNameAttribute:   "name",
		GroupMemberAttribute: "member",
		GroupMembersAreIds:   false,
		GroupQueryScope:      "sub",
	}

	OpenLdapSingleDomainTemplate = api.SLDAPIdpConfigOptions{
		UserObjectclass:      "person",
		UserIdAttribute:      "uid",
		UserNameAttribute:    "uid",
		UserEnabledAttribute: "nsAccountLock",
		UserEnabledDefault:   "FALSE",
		UserEnabledInvert:    true,
		UserAdditionalAttribute: []string{
			"displayName:displayname",
			"mobile:mobile",
			"mail:email",
		},
		UserQueryScope:       "sub",
		GroupObjectclass:     "ipausergroup",
		GroupIdAttribute:     "cn",
		GroupNameAttribute:   "cn",
		GroupMemberAttribute: "member",
		GroupMembersAreIds:   false,
		GroupQueryScope:      "sub",
	}
)
