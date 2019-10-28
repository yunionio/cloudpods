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

type PasswordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthContext struct {
	Source string `json:"source"`
	Ip     string `json:"ip"`
}

type AuthV2Input struct {
	PasswordCredentials PasswordCredentials `json:"password_credentials"`
	TenantName          string              `json:"tenant_name"`
	TenantId            string              `json:"tenant_id"`
	Token               struct {
		Id string `json:"id"`
	} `json:"token"`
	Context AuthContext `json:"context"`
}

type TenantV2 struct {
	SIdentityBaseResource
}

type TokenV2 struct {
	Token struct {
		Id      string     `json:"id"`
		Expires time.Time  `json:"expires"`
		Tenant  []TenantV2 `json:"tenants"`
	} `json:"token"`
}
