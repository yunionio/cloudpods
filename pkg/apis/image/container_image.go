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

const (
	CONTAINER_IMAGE_STATUS_READY = "ready"
)

type ContainerImageListInput struct {
	apis.SharableVirtualResourceListInput

	ImageLabel string `json:"image_label"`
	ImageName  string `json:"image_name"`
	RegistryId string `json:"registry_id"`
}

type ContainerImageCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName    string `json:"image_name"`
	ImageLabel   string `json:"image_label"`
	CredentialId string `json:"credential_id"`
	RegistryId   string `json:"registry_id"`
}

type ContainerImageUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName    *string `json:"image_name,omitempty"`
	ImageLabel   *string `json:"image_label,omitempty"`
	CredentialId *string `json:"credential_id,omitempty"`
	RegistryId   *string `json:"registry_id,omitempty"`
}

type ContainerImageDetails struct {
	apis.SharableVirtualResourceDetails

	Registry string `json:"registry"`
}
