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

package azure

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	compute_api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAMLProvider struct {
	multicloud.SResourceBase
	AzureTags
	client *SAzureClient

	Name     string
	Metadata samlutils.EntityDescriptor
}

func (self *SAMLProvider) Delete() error {
	return nil
}

func (self *SAMLProvider) GetGlobalId() string {
	return self.client.cpcfg.Id
}

func (self *SAMLProvider) GetId() string {
	return self.client.cpcfg.Id
}

func (self *SAMLProvider) GetName() string {
	return self.Name
}

func (self *SAMLProvider) GetStatus() string {
	return api.SAML_PROVIDER_STATUS_AVAILABLE
}

func (self *SAMLProvider) UpdateMetadata(metadata samlutils.EntityDescriptor) error {
	return nil
}

func (self *SAMLProvider) GetMetadataDocument() (*samlutils.EntityDescriptor, error) {
	return &self.Metadata, nil
}

func (self *SAMLProvider) GetAuthUrl(apiServer string) string {
	input := samlutils.SIdpInitiatedLoginInput{
		EntityID: cloudprovider.SAML_ENTITY_ID_AZURE,
		IdpId:    self.client.cpcfg.AccountId,
	}
	if self.client.GetAccessEnv() != compute_api.CLOUD_ACCESS_ENV_AZURE_GLOBAL {
		return ""
	}
	return httputils.JoinPath(apiServer, cloudid.SAML_IDP_PREFIX, fmt.Sprintf("sso?%s", jsonutils.Marshal(input).QueryString()))
}

func (self *SAzureClient) ListSAMLProviders() ([]SAMLProvider, error) {
	_, err := self.msGraphRequest("GET", "identityProviders", nil)
	if err != nil {
		return nil, err
	}
	return []SAMLProvider{}, nil
}

func (self *SAzureClient) InviteUser(email string) (*SClouduser, error) {
	body := jsonutils.Marshal(map[string]string{
		"invitedUserEmailAddress": email,
		"inviteRedirectUrl":       fmt.Sprintf("https://portal.azure.com/%s?login_hint=%s", self.tenantId, email),
	})
	resp, err := self.msGraphRequest("POST", "invitations", body)
	if err != nil {
		return nil, errors.Wrapf(err, "msGraphRequest.invitations")
	}
	inviteUrl, _ := resp.GetString("inviteRedeemUrl")
	err = cloudprovider.Wait(time.Second*2, time.Minute, func() (bool, error) {
		users, err := self.ListGraphUsers()
		if err != nil {
			return false, errors.Wrapf(err, "GetCloudusers")
		}
		for i := range users {
			users[i].inviteRedeemUrl = inviteUrl
			if users[i].GetEmailAddr() == email {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after invite %s", email)
	}
	users, err := self.ListGraphUsers()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudusers")
	}
	for i := range users {
		users[i].inviteRedeemUrl = inviteUrl
		if users[i].GetEmailAddr() == email {
			return &users[i], nil
		}
	}

	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after invite %s", email)
}

func (self *SAzureClient) CreateSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (*SAMLProvider, error) {
	return &SAMLProvider{
		client:   self,
		Name:     opts.Name,
		Metadata: opts.Metadata,
	}, nil
}
