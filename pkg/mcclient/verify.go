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

func (this *Client) AuthenticateVerify(uid, contactType, verifyCode string, projectId, projectName, projectDomain string, cliIp string) (TokenCredential, error) {
	aCtx := SAuthContext{
		// OpenID Connect auth must comes from Web
		Source: AuthSourceWeb,
		Ip:     cliIp,
	}
	return this.authenticateVerifyWithContext(uid, contactType, verifyCode, projectId, projectName, projectDomain, aCtx)
}

func (this *Client) authenticateVerifyWithContext(uid, contactType, verifyCode string, projectId, projectName, projectDomain string, aCtx SAuthContext) (TokenCredential, error) {
	if this.AuthVersion() != "v3" {
		return nil, httperrors.ErrNotSupported
	}
	input := SAuthenticationInputV3{}
	input.Auth.Identity.Methods = []string{api.AUTH_METHOD_VERIFY}
	input.Auth.Identity.Verify.Uid = uid
	input.Auth.Identity.Verify.ContactType = contactType
	input.Auth.Identity.Verify.VerifyCode = verifyCode
	if len(projectId) > 0 {
		input.Auth.Scope.Project.Id = projectId
	}
	if len(projectName) > 0 {
		input.Auth.Scope.Project.Name = projectName
		if len(projectDomain) > 0 {
			input.Auth.Scope.Project.Domain.Name = projectDomain
		}
	}
	input.Auth.Context = aCtx
	return this._authV3Input(input)
}
