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

package qcloud

import (
	"encoding/base64"
	"fmt"
	"time"
	"unicode"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAMLProvider struct {
	multicloud.SResourceBase
	QcloudTags
	client *SQcloudClient

	Name         string
	Description  string
	CreateTime   time.Time
	ModifyTime   time.Time
	SAMLMetadata string
}

func (self *SAMLProvider) GetId() string {
	return self.Name
}

func (self *SAMLProvider) GetGlobalId() string {
	return self.Name
}

func (self *SAMLProvider) GetName() string {
	return self.Name
}

func (self *SAMLProvider) GetStatus() string {
	return api.SAML_PROVIDER_STATUS_AVAILABLE
}

func (self *SAMLProvider) Delete() error {
	return self.client.DeleteSAMLProvider(self.Name)
}

func (self *SAMLProvider) GetAuthUrl(apiServer string) string {
	return fmt.Sprintf("https://cloud.tencent.com/login/forwardIdp/%s/%s", self.client.ownerName, self.Name)
}

func (self *SAMLProvider) GetMetadataDocument() (*samlutils.EntityDescriptor, error) {
	provider, err := self.client.GetSAMLProvider(self.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProvider(%s)", self.Name)
	}
	metadata, err := base64.StdEncoding.DecodeString(provider.SAMLMetadata)
	if err != nil {
		return nil, errors.Wrapf(err, "decode metadata")
	}
	ret, err := samlutils.ParseMetadata(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseMetadata")
	}
	return &ret, nil
}

func (self *SAMLProvider) UpdateMetadata(metadata samlutils.EntityDescriptor) error {
	return self.client.UpdateSAMLProvider(self.Name, metadata.String(), "")
}

func (self *SQcloudClient) ListSAMLProviders() ([]SAMLProvider, error) {
	resp, err := self.camRequest("ListSAMLProviders", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "ListSAMLProviders")
	}
	result := []SAMLProvider{}
	err = resp.Unmarshal(&result, "SAMLProviderSet")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return result, nil
}

func (self *SQcloudClient) CreateSAMLProvider(name, metadata, desc string) (*SAMLProvider, error) {
	if len(desc) == 0 {
		desc = "For CloudId Service"
	}
	//支持3-128个数字、大小写字母、和+=,.@_-
	name = func() string {
		ret := ""
		for _, c := range name {
			if unicode.IsLetter(c) || unicode.IsNumber(c) ||
				c == '+' || c == '=' || c == ',' || c == '.' || c == '@' || c == '_' || c == '-' {
				ret += string(c)
			} else {
				ret += "-"
			}
		}
		return ret
	}()
	if len(name) > 128 {
		name = name[:128]
	}
	params := map[string]string{
		"Name":                 name,
		"Description":          desc,
		"SAMLMetadataDocument": base64.StdEncoding.EncodeToString([]byte(metadata)),
	}
	_, err := self.camRequest("CreateSAMLProvider", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSAMLProvider")
	}
	return self.GetSAMLProvider(name)
}

func (self *SQcloudClient) GetSAMLProvider(name string) (*SAMLProvider, error) {
	params := map[string]string{
		"Name": name,
	}
	resp, err := self.camRequest("GetSAMLProvider", params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProvider")
	}
	result := &SAMLProvider{client: self}
	err = resp.Unmarshal(result)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return result, nil
}

func (self *SQcloudClient) DeleteSAMLProvider(name string) error {
	params := map[string]string{
		"Name": name,
	}
	_, err := self.camRequest("DeleteSAMLProvider", params)
	return err
}

func (self *SQcloudClient) UpdateSAMLProvider(name, metadata, desc string) error {
	params := map[string]string{
		"Name": name,
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	if len(metadata) > 0 {
		params["SAMLMetadataDocument"] = base64.StdEncoding.EncodeToString([]byte(metadata))
	}
	_, err := self.camRequest("UpdateSAMLProvider", params)
	if err != nil {
		return errors.Wrap(err, "UpdateSAMLProvider")
	}
	return nil
}

func (self *SQcloudClient) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
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
