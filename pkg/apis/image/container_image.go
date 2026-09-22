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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	CONTAINER_IMAGE_STATUS_READY = "ready"
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&ContainerImageEnvs{}), func() gotypes.ISerializable {
		return &ContainerImageEnvs{}
	})
}

// ContainerImageEnvs is the default environment variables of a container image.
// It is a named serializable type so that it can be stored as a compound column.
type ContainerImageEnvs []*apis.ContainerKeyValue

func (e ContainerImageEnvs) String() string {
	return jsonutils.Marshal(e).String()
}

func (e ContainerImageEnvs) IsZero() bool {
	return len(e) == 0
}

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

	// Default startup command when a container does not specify one
	Command []string `json:"command"`
	// Default startup args when a container does not specify one
	Args []string `json:"args"`
	// Default environment variables, appended to the container envs
	Envs *ContainerImageEnvs `json:"envs"`
}

type ContainerImageUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName    *string `json:"image_name,omitempty"`
	ImageLabel   *string `json:"image_label,omitempty"`
	CredentialId *string `json:"credential_id,omitempty"`
	RegistryId   *string `json:"registry_id,omitempty"`

	Command *[]string           `json:"command,omitempty"`
	Args    *[]string           `json:"args,omitempty"`
	Envs    *ContainerImageEnvs `json:"envs,omitempty"`
}

type ContainerImageDetails struct {
	apis.SharableVirtualResourceDetails

	Registry string `json:"registry"`
}
