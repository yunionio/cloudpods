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

package huawei

import (
	"fmt"
	"time"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/samlutils"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAMLProviderLinks struct {
	Self      string
	Protocols string
}

type SAMLProvider struct {
	multicloud.SResourceBase
	HuaweiTags
	client *SHuaweiClient

	Id          string
	Links       SAMLProviderLinks
	Description string
}

func (self *SAMLProvider) GetId() string {
	return self.Id
}

func (self *SAMLProvider) GetGlobalId() string {
	return self.Id
}

func (self *SAMLProvider) GetName() string {
	return self.Id
}

func (self *SAMLProvider) GetStatus() string {
	mapping, _ := self.client.findMapping()
	if mapping != nil {
		return api.SAML_PROVIDER_STATUS_AVAILABLE
	}
	return api.SAML_PROVIDER_STATUS_UNVALIABLE
}

func (self *SAMLProvider) GetAuthUrl(apiServer string) string {
	return fmt.Sprintf("https://auth.huaweicloud.com/authui/federation/websso?domain_id=%s&idp=%s&protocol=saml", self.client.ownerId, self.Id)
}

func (self *SAMLProvider) Delete() error {
	return self.client.DeleteSAMLProvider(self.Id)
}

func (self *SAMLProvider) GetMetadataDocument() (*samlutils.EntityDescriptor, error) {
	info, err := self.client.GetSAMLProviderMetadata(self.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProviderMetadata(%s)", self.Id)
	}
	metadata, err := samlutils.ParseMetadata([]byte(info.Data))
	if err != nil {
		return nil, errors.Wrapf(err, "ParseMetadata")
	}
	return &metadata, nil
}

func (self *SAMLProvider) UpdateMetadata(metadata samlutils.EntityDescriptor) error {
	return self.client.UpdateSAMLProviderMetadata(self.Id, metadata.String())
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListIdentityProviders
func (self *SHuaweiClient) ListSAMLProviders() ([]SAMLProvider, error) {
	resp, err := self.list(SERVICE_IAM_V3, "", "OS-FEDERATION/identity_providers", nil)
	if err != nil {
		return nil, err
	}
	ret := []SAMLProvider{}
	err = resp.Unmarshal(&ret, "identity_providers")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type SAMLProviderProtocol struct {
	MappingId string
	Id        string
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneShowProtocol
func (self *SHuaweiClient) GetSAMLProviderProtocols(id string) ([]SAMLProviderProtocol, error) {
	resp, err := self.list(SERVICE_IAM_V3, "", fmt.Sprintf("OS-FEDERATION/identity_providers/%s/protocols", id), nil)
	if err != nil {
		return nil, err
	}
	ret := []SAMLProviderProtocol{}
	err = resp.Unmarshal(&ret, "protocols")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneDeleteProtocol
func (self *SHuaweiClient) DeleteSAMLProviderProtocol(spId, id string) error {
	res := fmt.Sprintf("OS-FEDERATION/identity_providers/%s/protocols/%s", spId, id)
	_, err := self.delete(SERVICE_IAM_V3, "", res)
	return err
}

type SAMLProviderMetadata struct {
	DomainId     string
	UpdateTime   time.Time
	Data         string
	IdpId        string
	ProtocolId   string
	Id           string
	EntityId     string
	XaccountType string
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=ShowMetadata
func (self *SHuaweiClient) GetSAMLProviderMetadata(id string) (*SAMLProviderMetadata, error) {
	resp, err := self.list(SERVICE_IAM_V3_EXT, "", fmt.Sprintf("OS-FEDERATION/identity_providers/%s/protocols/saml/metadata", id), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "show metadata")
	}
	metadata := &SAMLProviderMetadata{}
	err = resp.Unmarshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return metadata, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=CreateMetadata
func (self *SHuaweiClient) UpdateSAMLProviderMetadata(id, metadata string) error {
	params := map[string]interface{}{
		"domain_id":     self.ownerId,
		"xaccount_type": "",
		"metadata":      metadata,
	}
	_, err := self.post(SERVICE_IAM_V3_EXT, "", fmt.Sprintf("OS-FEDERATION/identity_providers/%s/protocols/saml/metadata", id), params)
	return err
}

func (self *SHuaweiClient) GetICloudSAMLProviders() ([]cloudprovider.ICloudSAMLProvider, error) {
	samls, err := self.ListSAMLProviders()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSAMLProviders")
	}
	ret := []cloudprovider.ICloudSAMLProvider{}
	for i := range samls {
		samls[i].client = self
		ret = append(ret, &samls[i])
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneDeleteIdentityProvider
func (self *SHuaweiClient) DeleteSAMLProvider(id string) error {
	_, err := self.delete(SERVICE_IAM_V3, "", "OS-FEDERATION/identity_providers/"+id)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneCreateIdentityProvider
func (self *SHuaweiClient) CreateSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (*SAMLProvider, error) {
	params := map[string]interface{}{
		"identity_provider": map[string]interface{}{
			"description": opts.Name,
			"enabled":     true,
		},
	}
	name := []byte{}
	for _, c := range opts.Name {
		if unicode.IsLetter(c) || unicode.IsNumber(c) || c == '-' || c == '_' {
			name = append(name, byte(c))
		} else {
			name = append(name, '-')
		}
	}
	samlName := string(name)
	var err error
	err = func() error {
		idx := 1
		for {
			_, err = self.put(SERVICE_IAM_V3, "", "OS-FEDERATION/identity_providers/"+samlName, params)
			if err == nil {
				return nil
			}
			he, ok := errors.Cause(err).(*httputils.JSONClientError)
			if !ok {
				return errors.Wrapf(err, "SAMLProviders.Update")
			}
			if he.Code != 409 {
				return errors.Wrapf(err, "SAMLProviders.Update")
			}
			samlName = fmt.Sprintf("%s-%d", string(name), idx)
			idx++
			if idx >= 40 {
				break
			}
		}
		return err
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "saml provider create")
	}
	ret := SAMLProvider{client: self, Id: samlName}
	err = self.UpdateSAMLProviderMetadata(samlName, opts.Metadata.String())
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	err = self.InitSAMLProviderMapping(samlName)
	if err != nil {
		return nil, errors.Wrapf(err, "InitSAMLProviderMapping")
	}
	return &ret, nil
}

type SAMLProviderMapping struct {
	Id    string
	Rules jsonutils.JSONObject
}

var (
	onecloudMappingRules = jsonutils.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"remote": []map[string]interface{}{
					{
						"type": "User",
					},
					{
						"type": "Groups",
					},
				},
				"local": []map[string]interface{}{
					{
						"groups": "{1}",
						"user":   map[string]string{"name": "{0}"},
					},
				},
			},
		},
	})
)

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListMappings
func (self *SHuaweiClient) ListSAMLProviderMappings() ([]SAMLProviderMapping, error) {
	resp, err := self.list(SERVICE_IAM_V3, "", "OS-FEDERATION/mappings", nil)
	if err != nil {
		return nil, err
	}
	ret := []SAMLProviderMapping{}
	err = resp.Unmarshal(&ret, "mappings")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SHuaweiClient) findMapping() (*SAMLProviderMapping, error) {
	mappings, err := self.ListSAMLProviderMappings()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSAMLProviderMappings")
	}
	for i := range mappings {
		if jsonutils.Marshal(map[string]interface{}{"rules": mappings[i].Rules}).Equals(jsonutils.Marshal(onecloudMappingRules)) {
			return &mappings[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneCreateMapping
func (self *SHuaweiClient) InitSAMLProviderMapping(spId string) error {
	mapping, err := self.findMapping()
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			return errors.Wrapf(err, "findMapping")
		}
		mappingId := stringutils.UUID4()
		params := map[string]interface{}{
			"mapping": onecloudMappingRules,
		}
		_, err := self.put(SERVICE_IAM_V3, "", "OS-FEDERATION/mappings/"+mappingId, params)
		if err != nil {
			return errors.Wrapf(err, "create mapping")
		}
		mapping = &SAMLProviderMapping{
			Id:    mappingId,
			Rules: onecloudMappingRules,
		}
	}
	protocols, err := self.GetSAMLProviderProtocols(spId)
	if err != nil {
		return errors.Wrapf(err, "GetSAMLProviderProtocols")
	}
	params := map[string]interface{}{
		"protocol": map[string]string{
			"mapping_id": mapping.Id,
		},
	}
	for i := range protocols {
		if protocols[i].Id == "saml" {
			if protocols[i].MappingId == mapping.Id {
				return nil
			}
			// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneUpdateMapping
			_, err := self.patch(SERVICE_IAM_V3, "", fmt.Sprintf("OS-FEDERATION/identity_providers/%s/protocols/saml", spId), nil, params)
			return err
		}
	}
	_, err = self.patch(SERVICE_IAM, "", fmt.Sprintf("OS-FEDERATION/identity_providers/%s/protocols/saml", spId), nil, params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneDeleteMapping
func (self *SHuaweiClient) DeleteSAMLProviderMapping(id string) error {
	_, err := self.delete(SERVICE_IAM_V3, "", "OS-FEDERATION/mappings/"+id)
	return err
}
