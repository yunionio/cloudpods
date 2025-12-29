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

import "time"

type SUserExtended struct {
	Id               string    `json:"id"`
	Name             string    `json:"name"`
	Enabled          bool      `json:"enabled"`
	ExpiredAt        time.Time `json:"expired_at"`
	DefaultProjectId string    `json:"default_project_id"`
	CreatedAt        time.Time `json:"created_at"`
	LastActiveAt     time.Time `json:"last_active_at"`
	DomainId         string    `json:"domain_id"`

	IsSystemAccount bool `json:"is_system_account"`

	Displayname string `json:"displayname"`
	Email       string `json:"email"`
	Mobile      string `json:"mobile"`

	LocalId              int    `json:"local_id"`
	LocalName            string `json:"local_name"`
	LocalFailedAuthCount int    `json:"local_failed_auth_count"`
	DomainName           string `json:"domain_name"`
	DomainEnabled        bool   `json:"domain_enabled"`
	IsLocal              bool   `json:"is_local"`
	// IdpId         string
	// IdpName       string

	AuditIds []string `json:"audit_ids"`
}
