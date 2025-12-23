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

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func (client *Client) AuthenticateAssume(token string, userId, projectId string, cliIp string) (TokenCredential, error) {
	aCtx := SAuthContext{
		// Assume auth must comes from API
		Source: AuthSourceAPI,
		Ip:     cliIp,
	}
	return client.authenticateAssumeWithContext(token, userId, projectId, aCtx)
}

func (client *Client) authenticateAssumeWithContext(token string, userId, projectId string, aCtx SAuthContext) (TokenCredential, error) {
	if client.AuthVersion() != "v3" {
		return nil, httperrors.ErrNotSupported
	}
	input := SAuthenticationInputV3{}
	input.Auth.Identity.Token.Id = token
	input.Auth.Identity.Methods = []string{api.AUTH_METHOD_ASSUME}
	input.Auth.Identity.Assume.User.Id = userId
	input.Auth.Scope.Project.Id = projectId
	input.Auth.Context = aCtx
	return client._authV3Input(input)
}
