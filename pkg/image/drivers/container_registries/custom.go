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

package container_registries

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/drivers/container_registries/client"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterContainerRegistryDriver(newCustomImpl())
}

func newCustomImpl() models.IContainerRegistryDriver {
	return new(customImpl)
}

type customImpl struct{}

func (c customImpl) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryTypeCustom
}

func (c customImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
	return data, nil
}

func (c customImpl) CreateCredential(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (string, error) {
	config := data.Config.Custom
	if config == nil {
		return "", nil
	}
	return createContainerImageSecret(ctx, userCred, ownerId, data.Name, &config.ContainerRegistryConfigCommon)
}

func (c customImpl) PreparePushImage(ctx context.Context, urlStr string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error {
	return httperrors.NewNotSupportedError("custom not support prepare push image")
}

func (c customImpl) DownloadImage(ctx context.Context, urlStr string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error) {
	return "", httperrors.NewNotSupportedError("custom not support download image")
}

func (c customImpl) GetDockerRegistryClient(urlStr string, config *api.ContainerRegistryConfig) (client.Client, error) {
	return client.NewCustomClient(), nil
}
