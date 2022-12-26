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

package aliyun

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAMLProvider struct {
	multicloud.SResourceBase
	AliyunTags
	client *SAliyunClient

	Arn              string
	CreateDate       time.Time
	Description      string
	SAMLProviderName string
	UpdateDate       time.Time

	// base64
	EncodedSAMLMetadataDocument string
}

func (self *SAMLProvider) GetName() string {
	return self.SAMLProviderName
}

func (self *SAMLProvider) GetGlobalId() string {
	return self.Arn
}

func (self *SAMLProvider) GetId() string {
	return self.Arn
}

func (self *SAMLProvider) GetAuthUrl(apiServer string) string {
	input := samlutils.SIdpInitiatedLoginInput{
		EntityID: cloudprovider.SAML_ENTITY_ID_ALIYUN_ROLE,
		IdpId:    self.client.cpcfg.AccountId,
	}
	return httputils.JoinPath(apiServer, cloudid.SAML_IDP_PREFIX, fmt.Sprintf("sso?%s", jsonutils.Marshal(input).QueryString()))
}

func (self *SAMLProvider) Delete() error {
	return self.client.DeleteSAMLProvider(self.SAMLProviderName)
}

func (self *SAMLProvider) GetStatus() string {
	return api.SAML_PROVIDER_STATUS_AVAILABLE
}

func (self *SAliyunClient) ListSAMLProviders(marker string, maxItems int) ([]SAMLProvider, string, error) {
	if maxItems < 1 || maxItems > 100 {
		maxItems = 100
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	resp, err := self.imsRequest("ListSAMLProviders", params)
	if err != nil {
		return nil, "", errors.Wrapf(err, "ListSAMLProviders")
	}
	result := []SAMLProvider{}
	err = resp.Unmarshal(&result, "SAMLProviders", "SAMLProvider")
	if err != nil {
		return nil, "", errors.Wrapf(err, "resp.Unmarshal")
	}
	marker, _ = resp.GetString("Marker")
	return result, marker, nil
}

func (self *SAliyunClient) DeleteSAMLProvider(name string) error {
	params := map[string]string{
		"SAMLProviderName": name,
	}
	_, err := self.imsRequest("DeleteSAMLProvider", params)
	if err != nil {
		return errors.Wrapf(err, "DeleteSAMLProvider")
	}
	return nil
}

func (self *SAMLProvider) GetMetadataDocument() (*samlutils.EntityDescriptor, error) {
	sp, err := self.client.GetSAMLProvider(self.SAMLProviderName)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProvider(%s)", self.SAMLProviderName)
	}
	metadata, err := base64.StdEncoding.DecodeString(sp.EncodedSAMLMetadataDocument)
	if err != nil {
		return nil, errors.Wrapf(err, "DecodeString")
	}
	ret, err := samlutils.ParseMetadata(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseMetadata")
	}
	return &ret, nil
}

func (self *SAMLProvider) UpdateMetadata(metadata samlutils.EntityDescriptor) error {
	return self.client.UpdateSAMLProvider(self.SAMLProviderName, metadata.String())
}

func (self *SAliyunClient) GetSAMLProvider(name string) (*SAMLProvider, error) {
	params := map[string]string{
		"SAMLProviderName": name,
	}
	resp, err := self.imsRequest("GetSAMLProvider", params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProvider")
	}
	sp := &SAMLProvider{client: self}
	err = resp.Unmarshal(sp, "SAMLProvider")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return sp, nil
}

func (self *SAliyunClient) UpdateSAMLProvider(name string, metadata string) error {
	params := map[string]string{
		"SAMLProviderName":               name,
		"NewEncodedSAMLMetadataDocument": base64.StdEncoding.EncodeToString([]byte(metadata)),
	}
	_, err := self.imsRequest("UpdateSAMLProvider", params)
	if err != nil {
		return errors.Wrapf(err, "GetSAMLProvider")
	}
	return nil
}

func (self *SAliyunClient) CreateSAMLProvider(name, metadata, desc string) (*SAMLProvider, error) {
	name = strings.ReplaceAll(name, ":", "_")
	params := map[string]string{
		"EncodedSAMLMetadataDocument": base64.StdEncoding.EncodeToString([]byte(metadata)),
		"SAMLProviderName":            name,
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	resp, err := self.imsRequest("CreateSAMLProvider", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSAMLProvider")
	}
	sp := &SAMLProvider{client: self}
	err = resp.Unmarshal(sp, "SAMLProvider")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return sp, nil
}

func (self *SAliyunClient) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	marker := ""
	sps := []SAMLProvider{}
	for {
		part, _marker, err := self.ListSAMLProviders(marker, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "ListSAMLProviders")
		}
		sps = append(sps, part...)
		if len(_marker) == 0 {
			break
		}
		marker = _marker
	}
	ret := []cloudprovider.ICloudSAMLProvider{}
	for i := range sps {
		sps[i].client = self
		ret = append(ret, &sps[i])
	}
	return ret, nil
}
