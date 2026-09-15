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

package client

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/image"
)

type customClient struct{}

func NewCustomClient() Client {
	return &customClient{}
}

func (c *customClient) Ping(ctx context.Context) error {
	return nil
}

func (c *customClient) ListImages(ctx context.Context, input *api.ContainerRegistryListImagesInput) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (c *customClient) ListImageTags(ctx context.Context, image string) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (c *customClient) AnalysisImageTarMetadata(tarPath string) (*ImageMetadata, error) {
	return nil, nil
}

func (c *customClient) PushImage(ctx context.Context, input *ImageMetadata, tarPath string) error {
	return nil
}
