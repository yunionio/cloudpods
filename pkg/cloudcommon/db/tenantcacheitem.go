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

package db

import (
	"time"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
)

type SCachedTenant struct {
	Id               string            `json:"id"`
	Name             string            `json:"name"`
	DomainId         string            `json:"domain_id"`
	ProjectDomain    string            `json:"project_domain"`
	Metadata         map[string]string `json:"metadata"`
	PendingDeleted   bool              `json:"pending_deleted"`
	PendingDeletedAt time.Time         `json:"pending_deleted_at"`
}

func (s SCachedTenant) objType() string {
	if s.DomainId == identityapi.KeystoneDomainRoot && s.ProjectDomain == identityapi.KeystoneDomainRoot {
		return "domain"
	} else {
		return "project"
	}
}

type SCachedUser struct {
	SCachedTenant
	Lang string
}
