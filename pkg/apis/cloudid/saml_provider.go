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

package cloudid

import (
	"yunion.io/x/cloudmux/pkg/apis/cloudid"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	SAML_PROVIDER_STATUS_AVAILABLE              = cloudid.SAML_PROVIDER_STATUS_AVAILABLE
	SAML_PROVIDER_STATUS_NOT_MATCH              = "not_match"
	SAML_PROVIDER_STATUS_DELETING               = "deleting"
	SAML_PROVIDER_STATUS_DELETE_FAILED          = "delete_failed"
	SAML_PROVIDER_STATUS_UNKNOWN                = "unknown"
	SAML_PROVIDER_STATUS_CREATING               = "creating"
	SAML_PROVIDER_STATUS_CREATE_FAILED          = "create_failed"
	SAML_PROVIDER_STATUS_UNVALIABLE             = cloudid.SAML_PROVIDER_STATUS_UNVALIABLE
	SAML_PROVIDER_STATUS_UPDATE_METADATA        = "update_metadata"
	SAML_PROVIDER_STATUS_UPDATE_METADATA_FAILED = "update_metadata_failed"
	SAML_PROVIDER_STATUS_SYNC                   = "sync"
)

type SAMLProviderListInput struct {
	apis.StatusInfrasResourceBaseListInput
	CloudaccountResourceListInput
}

type SAMLProviderDetails struct {
	apis.StatusInfrasResourceBaseDetails
	CloudaccountResourceDetails

	SSAMLProvider
}

type SAMLProviderCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	CloudaccountId string `json:"cloudaccount_id"`

	// swagger:ignore
	EntityId string `json:"entity_id"`

	// swagger:ignore
	MetadataDocument string `json:"metadata_document"`
}
