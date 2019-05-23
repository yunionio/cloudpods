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

type SOwnerId struct {
	Domain    string
	DomainId  string
	Project   string
	ProjectId string

	User         string
	UserId       string
	UserDomain   string
	UserDomainId string
}

func (o *SOwnerId) GetProjectId() string {
	return o.ProjectId
}

func (o *SOwnerId) GetUserId() string {
	return o.UserId
}

func (o *SOwnerId) GetTenantId() string {
	return o.ProjectId
}

func (o *SOwnerId) GetProjectDomainId() string {
	return o.DomainId
}

func (o *SOwnerId) GetUserName() string {
	return o.User
}

func (o *SOwnerId) GetProjectName() string {
	return o.Project
}

func (o *SOwnerId) GetTenantName() string {
	return o.Project
}

func (o *SOwnerId) GetProjectDomain() string {
	return o.Domain
}

func (o *SOwnerId) GetDomainId() string {
	return o.UserDomainId
}

func (o *SOwnerId) GetDomainName() string {
	return o.UserDomain
}
