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

package image

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type ContainerRegistryType string

const (
	ContainerRegistryTypeHarbor = "harbor"
	ContainerRegistryTypeCommon = "common"
	ContainerRegistryTypeCustom = "custom"
)

type ContainerRegistryListInput struct {
	apis.SharableVirtualResourceListInput

	Type string `json:"type"`
	Url  string `json:"url"`
}

type ContainerRegistryConfigCommon struct {
	apis.ContainerPullImageAuthConfig
}

type ContainerRegistryConfigHarbor struct {
	ContainerRegistryConfigCommon
}

type ContainerRegistryConfigCustom struct {
	ContainerRegistryConfigCommon
}

type ContainerRegistryConfig struct {
	Type   ContainerRegistryType          `json:"type"`
	Common *ContainerRegistryConfigCommon `json:"common"`
	Harbor *ContainerRegistryConfigHarbor `json:"harbor"`
	Custom *ContainerRegistryConfigCustom `json:"custom"`
}

type ContainerRegistryCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	// Repo type
	// required: true
	// enum: harbor,common,custom
	Type ContainerRegistryType `json:"type"`

	// Repo URL
	// required: true
	// example: https://10.127.190.187
	Url string `json:"url"`

	// Configuration info
	Config ContainerRegistryConfig `json:"config"`

	// Credential ID (used when importing existing credentials)
	CredentialId string `json:"credential_id"`
}

type ContainerRegistryUploadImageInput struct {
	// Repository is the path on server, e.g. 'yunion/influxdb'
	Repository string `json:"repository"`
	// Tag is image tag
	Tag string `json:"tag"`
}

type ContainerRegistryGetImageTagsInput struct {
	Repository string `json:"repository"`
}

type ContainerRegistryManagerDownloadImageInput struct {
	Insecure bool   `json:"insecure"`
	Image    string `json:"image"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ContainerRegistryDownloadImageInput struct {
	ImageName string `json:"image_name"`
	Tag       string `json:"tag"`
}

type ContainerRegistryListImagesInput struct {
	Details        bool   `json:"details"`
	RepositoryName string `json:"repository_name"`
}

type ContainerRegistryImportFromKubeserverInput struct {
	// Force re-import even if a registry with the same URL already exists
	Force bool `json:"force"`
}

type ContainerRegistryImportFromKubeserverOutput struct {
	Imported []string `json:"imported"`
	Skipped  []string `json:"skipped"`
	Failed   []string `json:"failed"`
}

type ContainerRegistryDetails struct {
	apis.SharableVirtualResourceDetails

	Url          string                   `json:"url"`
	Type         string                   `json:"type"`
	CredentialId string                   `json:"credential_id"`
	Credential   string                   `json:"credential"`
	Config       *ContainerRegistryConfig `json:"config"`
}
