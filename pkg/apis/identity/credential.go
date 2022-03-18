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

package identity

import "yunion.io/x/onecloud/pkg/apis"

type CredentialDetails struct {
	apis.StandaloneResourceDetails
	SCredential

	Blob     string `json:"blob"`
	User     string `json:"user"`
	Domain   string `json:"domain"`
	DomainId string `json:"domain_id"`
}

type CredentialUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	// enabled
	Enabled *bool `json:"enabled"`
}

type CredentialCreateInput struct {
	apis.StandaloneResourceCreateInput

	Type string `json:"type"`

	ProjectId string `json:"project_id"`

	UserId string `json:"user_id"`

	Blob string `json:"blob"`

	// Ignore
	EncryptedBlob string `json:"encrypted_blob"`

	// Ignore
	KeyHash string `json:"key_hash"`
}
