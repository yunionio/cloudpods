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

package hcso

import (
	"fmt"
	"time"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/modules"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

type SAMLProviderLinks struct {
	Self      string
	Protocols string
}

type SAMLProvider struct {
	multicloud.SResourceBase
	huawei.HuaweiTags
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
	return fmt.Sprintf("https://auth.%s/authui/federation/websso?domain_id=%s&idp=%s&protocol=saml", self.client.endpoints.EndpointDomain, self.client.ownerId, self.Id)
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

func (self *SHuaweiClient) ListSAMLProviders() ([]SAMLProvider, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrapf(err, "newGeneralAPIClient")
	}
	samls := []SAMLProvider{}
	err = doListAllWithNextLink(client.SAMLProviders.List, nil, &samls)
	if err != nil {
		return nil, errors.Wrapf(err, "doListAll")
	}
	return samls, nil
}

type SAMLProviderProtocol struct {
	MappingId string
	Id        string
}

func (self *SHuaweiClient) GetSAMLProviderProtocols(id string) ([]SAMLProviderProtocol, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	resp, err := client.SAMLProviders.ListInContextWithSpec(nil, fmt.Sprintf("%s/protocols", id), nil, "protocols")
	if err != nil {
		return nil, errors.Wrapf(err, "ListInContextWithSpec")
	}
	protocols := []SAMLProviderProtocol{}
	return protocols, jsonutils.Update(&protocols, resp.Data)
}

func (self *SHuaweiClient) DeleteSAMLProviderProtocol(spId, id string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	_, err = client.SAMLProviders.DeleteInContextWithSpec(nil, spId, fmt.Sprintf("protocols/%s", id), nil, nil, "")
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

func (self *SHuaweiClient) GetSAMLProviderMetadata(id string) (*SAMLProviderMetadata, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	client.SAMLProviders.SetVersion("v3-ext/OS-FEDERATION")
	resp, err := client.SAMLProviders.GetInContextWithSpec(nil, id, fmt.Sprintf("protocols/saml/metadata"), nil, "")
	if err != nil {
		return nil, err
	}

	metadata := &SAMLProviderMetadata{}
	err = resp.Unmarshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return metadata, nil
}

func (self *SHuaweiClient) UpdateSAMLProviderMetadata(id, metadata string) error {
	params := map[string]string{
		"domain_id":     self.ownerId,
		"xaccount_type": "",
		"metadata":      metadata,
	}
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	client.SAMLProviders.SetVersion("v3-ext/OS-FEDERATION")
	_, err = client.SAMLProviders.PerformAction2("protocols/saml/metadata", id, jsonutils.Marshal(params), "")
	if err != nil {
		return errors.Wrapf(err, "SAMLProvider.PerformAction")
	}
	return nil
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

func (self *SHuaweiClient) DeleteSAMLProvider(id string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}
	_, err = client.SAMLProviders.Delete(id, nil)
	return err
}

func (self *SHuaweiClient) CreateSAMLProvider(opts *cloudprovider.SAMLProviderCreateOptions) (*SAMLProvider, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	params := jsonutils.Marshal(map[string]interface{}{
		"identity_provider": map[string]interface{}{
			"description": opts.Name,
			"enabled":     true,
		},
	})
	opts.Name = fmt.Sprintf("%s-%s", self.ownerName, opts.Name)
	name := []byte{}
	for _, c := range opts.Name {
		if unicode.IsLetter(c) || unicode.IsNumber(c) || c == '-' || c == '_' {
			name = append(name, byte(c))
		} else {
			name = append(name, '-')
		}
	}
	samlName := string(name)
	err = func() error {
		idx := 1
		for {
			_, err = client.SAMLProviders.Update(samlName, params)
			if err == nil {
				return nil
			}
			he, ok := err.(*modules.HuaweiClientError)
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

func (self *SHuaweiClient) ListSAMLProviderMappings() ([]SAMLProviderMapping, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	mappings := []SAMLProviderMapping{}
	err = doListAllWithNextLink(client.SAMLProviderMappings.List, nil, &mappings)
	if err != nil {
		return nil, err
	}
	return mappings, nil
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

func (self *SHuaweiClient) InitSAMLProviderMapping(spId string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}

	mapping, err := self.findMapping()
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			return errors.Wrapf(err, "findMapping")
		}
		mappingId := stringutils.UUID4()
		params := map[string]interface{}{
			"mapping": onecloudMappingRules,
		}
		_, err = client.SAMLProviderMappings.Update(mappingId, jsonutils.Marshal(params))
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
			_, err = client.SAMLProviders.PatchInContextWithSpec(nil, spId, "protocols/saml", jsonutils.Marshal(params), "")
			return err
		}
	}
	_, err = client.SAMLProviders.UpdateInContextWithSpec(nil, spId, "protocols/saml", jsonutils.Marshal(params), "")
	return err
}

func (self *SHuaweiClient) DeleteSAMLProviderMapping(id string) error {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return errors.Wrap(err, "newGeneralAPIClient")
	}

	_, err = client.SAMLProviderMappings.Delete(id, nil)
	return err
}
