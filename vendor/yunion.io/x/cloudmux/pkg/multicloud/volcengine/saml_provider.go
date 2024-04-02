// Copyright 2023 Yunion
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

package volcengine

import (
	"encoding/base64"
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/samlutils"
)

type SSamlProvider struct {
	multicloud.SResourceBase
	multicloud.STagBase
	client *SVolcEngineClient

	Trn                         string
	EncodedSAMLMetadataDocument string
	ProviderName                string
	SAMLProviderName            string
	IdpType                     string
	SSOType                     string
	Status                      string
	Description                 string
}

func (self *SSamlProvider) GetName() string {
	if len(self.SAMLProviderName) > 0 {
		return self.SAMLProviderName
	}
	return self.ProviderName
}

func (self *SSamlProvider) GetGlobalId() string {
	if len(self.SAMLProviderName) > 0 {
		return self.SAMLProviderName
	}
	return self.ProviderName
}

func (self *SSamlProvider) GetId() string {
	if len(self.SAMLProviderName) > 0 {
		return self.SAMLProviderName
	}
	return self.ProviderName
}

func (self *SSamlProvider) GetAuthUrl(apiServer string) string {
	input := samlutils.SIdpInitiatedLoginInput{
		EntityID: cloudprovider.SAML_ENTITY_ID_VOLC_ENGINE,
		IdpId:    self.client.cpcfg.AccountId,
	}
	return httputils.JoinPath(apiServer, cloudid.SAML_IDP_PREFIX, fmt.Sprintf("sso?%s", jsonutils.Marshal(input).QueryString()))
}

func (self *SSamlProvider) Delete() error {
	return self.client.DeleteSamlProvider(self.GetName())
}

func (self *SSamlProvider) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (self *SSamlProvider) GetMetadataDocument() (*samlutils.EntityDescriptor, error) {
	provider, err := self.client.GetSamlProvider(self.GetName())
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(provider.EncodedSAMLMetadataDocument)
	if err != nil {
		return nil, err
	}
	ret, err := samlutils.ParseMetadata(data)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (self *SSamlProvider) UpdateMetadata(metadata samlutils.EntityDescriptor) error {
	return self.client.UpdateSAMLProvider(self.GetName(), metadata.String())
}

func (self *SVolcEngineClient) GetSamlProviders() ([]SSamlProvider, error) {
	params := map[string]string{}
	params["Limit"] = "50"
	ret := []SSamlProvider{}
	offset := 0
	for {
		params["Offset"] = fmt.Sprintf("%d", offset)
		resp, err := self.iamRequest("", "ListSAMLProviders", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			SAMLProviders []SSamlProvider
			Total         int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SAMLProviders...)
		if len(ret) >= part.Total || len(part.SAMLProviders) == 0 {
			break
		}
		offset = len(ret)
	}
	return ret, nil
}

func (self *SVolcEngineClient) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	providers, err := self.GetSamlProviders()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSAMLProvider{}
	for i := range providers {
		providers[i].client = self
		ret = append(ret, &providers[i])
	}
	return ret, nil
}

func (self *SVolcEngineClient) DeleteSamlProvider(name string) error {
	params := map[string]string{
		"SAMLProviderName": name,
	}
	_, err := self.iamRequest("", "DeleteSAMLProvider", params)
	return err
}

func (self *SVolcEngineClient) CreateSAMLProvider(name, metadata, desc string) (*SSamlProvider, error) {
	name = strings.ReplaceAll(name, ":", "_")
	params := map[string]string{
		"EncodedSAMLMetadataDocument": base64.StdEncoding.EncodeToString([]byte(metadata)),
		"SAMLProviderName":            name,
		"SSOType":                     "1",
		"Status":                      "1",
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	resp, err := self.iamRequest("", "CreateSAMLProvider", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSAMLProvider")
	}
	sp := &SSamlProvider{client: self}
	err = resp.Unmarshal(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return sp, nil
}

func (self *SVolcEngineClient) GetSamlProvider(name string) (*SSamlProvider, error) {
	params := map[string]string{
		"SAMLProviderName": name,
	}
	resp, err := self.iamRequest("", "GetSAMLProvider", params)
	if err != nil {
		return nil, err
	}
	ret := &SSamlProvider{client: self}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil

}

func (self *SVolcEngineClient) UpdateSAMLProvider(name, metadata string) error {
	name = strings.ReplaceAll(name, ":", "_")
	params := map[string]string{
		"EncodedSAMLMetadataDocument": base64.StdEncoding.EncodeToString([]byte(metadata)),
		"SAMLProviderName":            name,
		"Status":                      "1",
	}
	_, err := self.iamRequest("", "UpdateSAMLProvider", params)
	return err
}
