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

import (
	"time"
)

type UserDetails struct {
	EnabledIdentityBaseResourceDetails
	// IdpResourceInfo

	SUser

	GroupCount        int       `json:"group_count"`
	ProjectCount      int       `json:"project_count"`
	CredentialCount   int       `json:"credential_count"`
	FailedAuthCount   int       `json:"failed_auth_count"`
	FailedAuthAt      time.Time `json:"failed_auth_at"`
	PasswordExpiresAt time.Time `json:"password_expires_at"`

	NeedResetPassword bool `json:"need_reset_password"`

	Idps []IdpResourceInfo `json:"idps"`

	IsLocal bool `json:"is_local"`

	ExternalResourceInfo

	Projects []SFetchDomainObjectWithMetadata `json:"projects"`
}
