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

package mcclient

const (
	AuthSourceWeb = "web"
	AuthSourceAPI = "api"
	AuthSourceCli = "cli"
	AuthSourceSrv = "srv"
)

type SAuthContext struct {
	Source string `json:"source,omitempty"`
	Ip     string `json:"ip,omitempty"`
}

type SAuthenticationInputV2 struct {
	Auth struct {
		PasswordCredentials struct {
			Username string `json:"username,omitempty"`
			Password string `json:"password,omitempty"`
		} `json:"passwordCredentials,omitempty"`
		TenantName string `json:"tenantName,omitempty"`
		TenantId   string `json:"tenantId,omitempty"`
		Token      struct {
			Id string
		} `json:"token,omitempty"`
		Context SAuthContext `json:"context,omitempty"`
	} `json:"auth,omitempty"`
}

type SAuthenticationIdentity struct {
	Methods  []string `json:"methods,omitempty"`
	Password struct {
		User struct {
			Id       string `json:"id,omitempty"`
			Name     string `json:"name,omitempty"`
			Password string `json:"password,omitempty"`
			Domain   struct {
				Id   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			}
		} `json:"user,omitempty"`
	} `json:"password,omitempty"`
	Token struct {
		Id string `json:"id,omitempty"`
	} `json:"token,omitempty"`
	AccessKeyRequest string `json:"access_key_secret,omitempty"`

	CASTicket struct {
		Id string `json:"id,omitempty"`
	} `json:"cas_ticket,omitempty"`
}

type SAuthenticationInputV3 struct {
	Auth struct {
		Identity SAuthenticationIdentity `json:"identity,omitempty"`
		Scope    struct {
			Project struct {
				Id     string `json:"id,omitempty"`
				Name   string `json:"name,omitempty"`
				Domain struct {
					Id   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				} `json:"domain,omitempty"`
			} `json:"project,omitempty"`
			Domain struct {
				Id   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			} `json:"domain,omitempty"`
		} `json:"scope,omitempty"`
		Context SAuthContext `json:"context,omitempty"`
	} `json:"auth,omitempty"`
}
