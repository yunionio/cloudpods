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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type UserListOptions struct {
	options.BaseListOptions
	Name                    string `help:"Filter by name"`
	OrderByDomain           string `help:"order by domain name" choices:"asc|desc"`
	Role                    string `help:"Filter by role"`
	RoleAssignmentDomainId  string `help:"filter role assignment domain"`
	RoleAssignmentProjectId string `help:"filter role assignment project"`
	IdpId                   string `help:"filter by idp_id"`
	IdpEntityId             string `help:"filter by idp_entity_id"`
}

func (opts *UserListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type UserIdOptions struct {
	ID string `help:"ID or name of user"`
}

func (opts *UserIdOptions) GetId() string {
	return opts.ID
}

type UserDetailOptions struct {
	UserIdOptions
	Domain string `help:"Domain"`
}

func (opts *UserDetailOptions) Params() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	if len(opts.Domain) > 0 {
		ret.Add(jsonutils.NewString(opts.Domain), "domain_id")
	}
	return ret, nil
}
