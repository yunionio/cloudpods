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

package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetContainerImageManager()
}

var containerImageManager *SContainerImageManager

func GetContainerImageManager() *SContainerImageManager {
	if containerImageManager != nil {
		return containerImageManager
	}
	containerImageManager = &SContainerImageManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SContainerImage{},
			"container_images_tbl",
			"container_image",
			"container_images",
		),
	}
	containerImageManager.SetVirtualObject(containerImageManager)
	return containerImageManager
}

type SContainerImageManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SContainerImage struct {
	db.SSharableVirtualResourceBase

	ImageName    string `width:"256" charset:"utf8" nullable:"false" list:"user" create:"required" update:"user"`
	ImageLabel   string `width:"128" charset:"utf8" nullable:"false" list:"user" create:"required" update:"user"`
	CredentialId string `width:"128" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	RegistryId   string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

	// 默认启动命令，创建容器时未指定 command 则使用该值
	Command []string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`
	// 默认启动参数，创建容器时未指定 args 则使用该值
	Args []string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`
	// 默认环境变量，创建容器时追加到容器环境变量中，同名以容器指定值为准
	Envs *api.ContainerImageEnvs `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`
}

func fetchImageCredential(ctx context.Context, userCred mcclient.TokenCredential, cid string) (*identityapi.CredentialDetails, error) {
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	credJson, err := identity.Credentials.Get(s, cid, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Credentials.Get")
	}
	details := identityapi.CredentialDetails{}
	err = credJson.Unmarshal(&details)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &details, nil
}

func normalizeRegistryHost(url string) string {
	host := strings.TrimSpace(url)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	return strings.TrimRight(host, "/")
}

func ensureImageNameMatchesRegistry(imageName string, reg *SContainerRegistry) error {
	if reg == nil {
		return nil
	}
	prefix := normalizeRegistryHost(reg.Url)
	if prefix == "" {
		return httperrors.NewInputParameterError("container registry %s has empty url", reg.GetId())
	}
	if imageName == prefix || strings.HasPrefix(imageName, prefix+"/") {
		return nil
	}
	return httperrors.NewInputParameterError("image_name %q must start with registry prefix %q", imageName, prefix)
}

func (man *SContainerImageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ContainerImageCreateInput) (*api.ContainerImageCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	if input.ImageName == "" {
		return input, httperrors.NewInputParameterError("image_name is required")
	}
	if input.ImageLabel == "" {
		return input, httperrors.NewInputParameterError("image_label is required")
	}

	if len(input.CredentialId) > 0 {
		cred, err := fetchImageCredential(ctx, userCred, input.CredentialId)
		if err != nil {
			return input, errors.Wrap(err, "fetchImageCredential")
		}
		input.CredentialId = cred.Id
	}

	if len(input.RegistryId) > 0 {
		obj, err := GetContainerRegistryManager().FetchByIdOrName(ctx, userCred, input.RegistryId)
		if err != nil {
			return input, errors.Wrapf(err, "fetch container registry %s", input.RegistryId)
		}
		input.RegistryId = obj.GetId()
		reg, ok := obj.(*SContainerRegistry)
		if !ok {
			return input, httperrors.NewGeneralError(errors.Errorf("invalid container registry object"))
		}
		if err := ensureImageNameMatchesRegistry(input.ImageName, reg); err != nil {
			return input, err
		}
		if len(input.CredentialId) == 0 && reg.CredentialId != "" {
			input.CredentialId = reg.CredentialId
		}
	}

	input.Status = api.CONTAINER_IMAGE_STATUS_READY
	return input, nil
}

func (man *SContainerImageManager) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ContainerImageUpdateInput) (*api.ContainerImageUpdateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	if input.CredentialId != nil && len(*input.CredentialId) > 0 {
		cred, err := fetchImageCredential(ctx, userCred, *input.CredentialId)
		if err != nil {
			return input, errors.Wrap(err, "fetchImageCredential")
		}
		input.CredentialId = &cred.Id
	}

	if input.RegistryId != nil && len(*input.RegistryId) > 0 {
		obj, err := GetContainerRegistryManager().FetchByIdOrName(ctx, userCred, *input.RegistryId)
		if err != nil {
			return input, errors.Wrapf(err, "fetch container registry %s", *input.RegistryId)
		}
		id := obj.GetId()
		input.RegistryId = &id
		reg, ok := obj.(*SContainerRegistry)
		if !ok {
			return input, httperrors.NewGeneralError(errors.Errorf("invalid container registry object"))
		}
		if input.ImageName != nil {
			if err := ensureImageNameMatchesRegistry(*input.ImageName, reg); err != nil {
				return input, err
			}
		}
		if input.CredentialId == nil || len(*input.CredentialId) == 0 {
			if reg.CredentialId != "" {
				input.CredentialId = &reg.CredentialId
			}
		}
	}

	return input, nil
}

func (image *SContainerImage) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ContainerImageUpdateInput) (api.ContainerImageUpdateInput, error) {
	if input.CredentialId != nil && len(*input.CredentialId) > 0 {
		cred, err := fetchImageCredential(ctx, userCred, *input.CredentialId)
		if err != nil {
			return input, errors.Wrap(err, "fetchImageCredential")
		}
		input.CredentialId = &cred.Id
	}

	registryId := image.RegistryId
	if input.RegistryId != nil {
		registryId = *input.RegistryId
	}
	if len(registryId) > 0 {
		obj, err := GetContainerRegistryManager().FetchByIdOrName(ctx, userCred, registryId)
		if err != nil {
			return input, errors.Wrapf(err, "fetch container registry %s", registryId)
		}
		id := obj.GetId()
		if input.RegistryId != nil {
			input.RegistryId = &id
		}
		reg, ok := obj.(*SContainerRegistry)
		if !ok {
			return input, httperrors.NewGeneralError(errors.Errorf("invalid container registry object"))
		}
		imageName := image.ImageName
		if input.ImageName != nil {
			imageName = *input.ImageName
		}
		if err := ensureImageNameMatchesRegistry(imageName, reg); err != nil {
			return input, err
		}
		if input.CredentialId == nil || len(*input.CredentialId) == 0 {
			if input.RegistryId != nil && reg.CredentialId != "" {
				input.CredentialId = &reg.CredentialId
			}
		}
	}
	return input, nil
}

func (man *SContainerImageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ContainerImageListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	if input.IsPublic != nil {
		if *input.IsPublic {
			q = q.IsTrue("is_public")
		} else {
			q = q.IsFalse("is_public")
		}
	}
	if len(input.ImageLabel) > 0 {
		q = q.Equals("image_label", input.ImageLabel)
	}
	if len(input.ImageName) > 0 {
		q = q.Equals("image_name", input.ImageName)
	}
	if len(input.RegistryId) > 0 {
		q = q.Equals("registry_id", input.RegistryId)
	}
	return q, nil
}

func (manager *SContainerImageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ContainerImageDetails {
	rows := make([]api.ContainerImageDetails, len(objs))
	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	registryIds := make([]string, 0)
	for i := range objs {
		image := objs[i].(*SContainerImage)
		if image.RegistryId != "" {
			registryIds = append(registryIds, image.RegistryId)
		}
	}
	registryNames, err := db.FetchIdNameMap2(GetContainerRegistryManager(), registryIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 container_registries: %v", err)
	}

	for i := range rows {
		rows[i] = api.ContainerImageDetails{
			SharableVirtualResourceDetails: virtRows[i],
		}
		image := objs[i].(*SContainerImage)
		if name, ok := registryNames[image.RegistryId]; ok {
			rows[i].Registry = name
		}
	}
	return rows
}

func (image *SContainerImage) ToContainerImage() string {
	return fmt.Sprintf("%s:%s", image.ImageName, image.ImageLabel)
}
