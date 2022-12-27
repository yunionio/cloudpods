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

package aws

import (
	"fmt"
	"strings"
	"time"
	"unicode"

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
	AwsTags
	client *SAwsClient

	SAMLMetadataDocument string    `xml:"SAMLMetadataDocument"`
	Arn                  string    `xml:"Arn"`
	ValidUntil           time.Time `xml:"ValidUntil"`
	CreateDate           time.Time `xml:"CreateDate"`
}

func (self *SAMLProvider) GetGlobalId() string {
	return self.Arn
}

func (self *SAMLProvider) GetId() string {
	return self.Arn
}

func (self *SAMLProvider) GetName() string {
	if info := strings.Split(self.Arn, "/"); len(info) > 0 {
		return info[len(info)-1]
	}
	return self.Arn
}

func (self *SAMLProvider) GetStatus() string {
	return api.SAML_PROVIDER_STATUS_AVAILABLE
}

func (self *SAMLProvider) Delete() error {
	return self.client.DeleteSAMLProvider(self.Arn)
}

func (self *SAMLProvider) GetAuthUrl(apiServer string) string {
	input := samlutils.SIdpInitiatedLoginInput{
		EntityID: cloudprovider.SAML_ENTITY_ID_AWS_CN,
		IdpId:    self.client.cpcfg.AccountId,
	}
	if self.client.GetAccessEnv() == compute_api.CLOUD_ACCESS_ENV_AWS_GLOBAL {
		input.EntityID = cloudprovider.SAML_ENTITY_ID_AWS
	}
	return httputils.JoinPath(apiServer, cloudid.SAML_IDP_PREFIX, fmt.Sprintf("sso?%s", jsonutils.Marshal(input).QueryString()))
}

func (self *SAMLProvider) GetMetadataDocument() (*samlutils.EntityDescriptor, error) {
	saml, err := self.client.GetSAMLProvider(self.Arn)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProvider(%s)", self.Arn)
	}
	metadata, err := samlutils.ParseMetadata([]byte(saml.SAMLMetadataDocument))
	if err != nil {
		return nil, errors.Wrapf(err, "ParseMetadata")
	}
	return &metadata, nil
}

func (self *SAMLProvider) UpdateMetadata(metadata samlutils.EntityDescriptor) error {
	_, err := self.client.UpdateSAMLProvider(self.Arn, metadata.String())
	return err
}

type SAMLProviders struct {
	SAMLProviderList []SAMLProvider `xml:"SAMLProviderList>member"`
}

func (self *SAwsClient) ListSAMLProviders() ([]SAMLProvider, error) {
	result := SAMLProviders{}
	err := self.iamRequest("ListSAMLProviders", nil, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "ListSAMLProviders")
	}
	return result.SAMLProviderList, nil
}

func (self *SAwsClient) GetSAMLProvider(arn string) (*SAMLProvider, error) {
	result := &SAMLProvider{client: self, Arn: arn}
	params := map[string]string{"SAMLProviderArn": arn}
	err := self.iamRequest("GetSAMLProvider", params, result)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProvider")
	}
	return result, nil
}

func (self *SAwsClient) DeleteSAMLProvider(arn string) error {
	params := map[string]string{"SAMLProviderArn": arn}
	return self.iamRequest("DeleteSAMLProvider", params, nil)
}

func (self *SAwsClient) CreateSAMLProvider(name, metadata string) (*SAMLProvider, error) {
	name = func() string {
		ret := ""
		for _, s := range name {
			if unicode.IsLetter(s) || unicode.IsNumber(s) || s == '.' || s == '_' || s == '-' {
				ret += string(s)
			} else {
				ret += "-"
			}
		}
		if len(ret) > 128 {
			ret = ret[:128]
		}
		return ret
	}()
	params := map[string]string{
		"Name":                 name,
		"SAMLMetadataDocument": metadata,
	}
	result := struct {
		SAMLProviderArn string `xml:"SAMLProviderArn"`
	}{}
	err := self.iamRequest("CreateSAMLProvider", params, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSAMLProvider")
	}
	return self.GetSAMLProvider(result.SAMLProviderArn)
}

func (self *SAwsClient) UpdateSAMLProvider(arn, metadata string) (*SAMLProvider, error) {
	params := map[string]string{
		"SAMLProviderArn":      arn,
		"SAMLMetadataDocument": metadata,
	}
	saml := &SAMLProvider{client: self}
	err := self.iamRequest("UpdateSAMLProvider", params, saml)
	if err != nil {
		return nil, errors.Wrapf(err, "UpdateSAMLProvider")
	}
	return saml, nil
}

func (self *SAwsClient) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	providers, err := self.ListSAMLProviders()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSAMLProviders")
	}
	ret := []cloudprovider.ICloudSAMLProvider{}
	for i := range providers {
		providers[i].client = self
		ret = append(ret, &providers[i])
	}
	return ret, nil
}
