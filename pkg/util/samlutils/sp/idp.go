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

package sp

import (
	"net/url"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

type SSAMLIdentityProvider struct {
	entityId       string
	redirectSsoUrl string
}

func NewSAMLIdp(entityId, redirectSsoUrl string) *SSAMLIdentityProvider {
	return &SSAMLIdentityProvider{
		entityId:       entityId,
		redirectSsoUrl: redirectSsoUrl,
	}
}

func NewSAMLIdpFromDescriptor(desc samlutils.EntityDescriptor) (*SSAMLIdentityProvider, error) {
	entityId := desc.EntityId
	if desc.IDPSSODescriptor == nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "missing IDPSSODescriptor")
	}
	redirectSsoUrl := findSSOUrl(desc, samlutils.BINDING_HTTP_REDIRECT)
	return NewSAMLIdp(entityId, redirectSsoUrl), nil
}

func (idp *SSAMLIdentityProvider) GetEntityId() string {
	return idp.entityId
}

func findSSOUrl(desc samlutils.EntityDescriptor, binding string) string {
	for _, v := range desc.IDPSSODescriptor.SingleSignOnServices {
		if v.Binding == binding {
			return v.Location
		}
	}
	return ""
}

func (idp *SSAMLIdentityProvider) getRedirectSSOUrl() string {
	return idp.redirectSsoUrl
}

func (idp *SSAMLIdentityProvider) IsValid() error {
	if len(idp.GetEntityId()) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "empty EntityID")
	}
	ssoUrlStr := idp.getRedirectSSOUrl()
	if len(ssoUrlStr) == 0 {
		return errors.Wrap(httperrors.ErrInvalidFormat, "empty redirect SSO URL")
	}
	_, err := url.Parse(ssoUrlStr)
	if err != nil {
		return errors.Wrapf(err, "invalid redirect SSO URL: %s", ssoUrlStr)
	}
	return nil
}
