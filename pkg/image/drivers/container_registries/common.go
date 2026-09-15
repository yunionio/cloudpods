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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/drivers/container_registries/client"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/image/utils/skopeo"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identitymodules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/pkg/errors"
)

func init() {
	models.RegisterContainerRegistryDriver(newCommonImpl())
}

func newCommonImpl() models.IContainerRegistryDriver {
	return new(commonImpl)
}

type commonImpl struct{}

func (c commonImpl) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryTypeCommon
}

func (c commonImpl) GetDockerRegistryClient(urlStr string, config *api.ContainerRegistryConfig) (client.Client, error) {
	if config.Common == nil {
		return client.NewClient(urlStr, client.DockerAuthConfig{
			Username: "",
			Password: "",
		})
	}
	return client.NewClient(urlStr, client.DockerAuthConfig{
		Username: config.Common.Username,
		Password: config.Common.Password,
	})
}

func (c commonImpl) GetPathPrefix(inputUrl string) (string, error) {
	regUrl, err := url.Parse(inputUrl)
	if err != nil {
		return "", httperrors.NewInputParameterError("Invalid url: %q", inputUrl)
	}
	if regUrl.Path == "" {
		return "", httperrors.NewInputParameterError("Url %q must with path prefix used for a repository namespace", inputUrl)
	}
	return regUrl.Path, nil
}

func (c commonImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
	if _, err := c.GetPathPrefix(data.Url); err != nil {
		return nil, err
	}
	config := data.Config.Common
	if config != nil && config.Username != "" && config.Password != "" {
		cli, err := c.GetDockerRegistryClient(data.Url, &data.Config)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Get docker registry client: %v", err)
		}
		if err := cli.Ping(ctx); err != nil {
			return nil, httperrors.NewInputParameterError("Ping docker registry %q: %v", data.Url, err)
		}
	}
	return data, nil
}

func (c commonImpl) CreateCredential(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (string, error) {
	config := data.Config.Common
	if config == nil {
		return "", nil
	}
	return createContainerImageSecret(ctx, userCred, ownerId, data.Name, config)
}

func createContainerImageSecret(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, name string, config *api.ContainerRegistryConfigCommon) (string, error) {
	if config == nil {
		return "", httperrors.NewInputParameterError("Configuration of common is nil")
	}
	if config.Username == "" {
		return "", httperrors.NewInputParameterError("Username is required")
	}
	if config.Password == "" {
		return "", httperrors.NewInputParameterError("Password is required")
	}
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	if s == nil {
		return "", errors.Errorf("Get user session nil")
	}
	obj, err := identitymodules.Credentials.CreateContainerImageSecret(s, ownerId.GetProjectId(), name, &identityapi.CredentialContainerImageBlob{
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		return "", err
	}
	return obj.GetString("id")
}

func (c commonImpl) PreparePushImage(ctx context.Context, urlStr string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error {
	parts := strings.Split(meta.Ref.Repository, "/")
	meta.Ref.Repository = parts[len(parts)-1]
	return nil
}

func (c commonImpl) DownloadImage(ctx context.Context, regUrl string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error) {
	srcUrl := strings.TrimPrefix(strings.TrimPrefix(regUrl, "http://"), "https://")
	imgName := fmt.Sprintf("%s:%s", input.ImageName, input.Tag)
	savedPath := fmt.Sprintf("/tmp/%s.tar", strings.ReplaceAll(imgName, ":", "-"))
	imageUrl := filepath.Join(srcUrl, imgName)
	copyParams := &skopeo.CopyParams{
		SrcTLSVerify: false,
		SrcUsername:  conf.Common.Username,
		SrcPassword:  conf.Common.Password,
		SrcPath:      imageUrl,
		TargetPath:   savedPath,
	}
	if err := skopeo.NewSkopeo().Copy(copyParams); err != nil {
		return "", errors.Wrapf(err, "copy container image to local path %s", savedPath)
	}
	gzFilePath := fmt.Sprintf("%s.gz", savedPath)
	f, err := os.Create(gzFilePath)
	if err != nil {
		return "", errors.Wrapf(err, "create %s", gzFilePath)
	}
	defer f.Close()
	w, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
	if err != nil {
		return "", errors.Wrap(err, "gzip.NewWriterLevel")
	}
	defer w.Close()
	savedFile, err := os.Open(savedPath)
	if err != nil {
		return "", errors.Wrapf(err, "open %s", savedPath)
	}
	defer savedFile.Close()
	if _, err := io.Copy(w, savedFile); err != nil {
		return "", errors.Wrapf(err, "gzip %s to %s", savedPath, gzFilePath)
	}
	w.Flush()
	return gzFilePath, nil
}
