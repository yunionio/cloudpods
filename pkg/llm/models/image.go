package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	imagemodules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetLLMImageManager()
}

var llmImageManager *SLLMImageManager

func GetLLMImageManager() *SLLMImageManager {
	if llmImageManager != nil {
		return llmImageManager
	}
	llmImageManager = &SLLMImageManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLLMImage{},
			"llm_images_tbl",
			"llm_image",
			"llm_images",
		),
	}
	llmImageManager.SetVirtualObject(llmImageManager)
	return llmImageManager
}

type SLLMImageManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SLLMImage struct {
	db.SSharableVirtualResourceBase

	ImageName  string `width:"256" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"user"`
	ImageLabel string `width:"128" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"user"`

	CredentialId     string                     `width:"128" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" update:"user"`
	LLMType          string                     `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"user"`
	AppName          string                     `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" json:"app_name"`
	DesktopConfig    *api.LLMImageDesktopConfig `json:"desktop_config" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`
	ImageSource      string                     `width:"32" charset:"ascii" nullable:"false" default:"manual" list:"user" create:"optional" update:"user"`
	ContainerImageId string                     `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
}

type containerImageMeta struct {
	Id           string
	Name         string
	ImageName    string
	ImageLabel   string
	CredentialId string
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

func fetchContainerImageMeta(ctx context.Context, userCred mcclient.TokenCredential, imageId string) (*containerImageMeta, error) {
	var s *mcclient.ClientSession
	if userCred != nil {
		s = auth.GetSession(ctx, userCred, options.Options.Region)
	} else {
		s = auth.GetAdminSession(ctx, options.Options.Region)
	}
	obj, err := imagemodules.ContainerImages.Get(s, imageId, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "get container_image %s", imageId))
	}
	meta := &containerImageMeta{}
	meta.Id, _ = obj.GetString("id")
	meta.Name, _ = obj.GetString("name")
	meta.ImageName, _ = obj.GetString("image_name")
	meta.ImageLabel, _ = obj.GetString("image_label")
	if meta.ImageName == "" || meta.ImageLabel == "" {
		return nil, httperrors.NewInputParameterError("container_image %s missing image_name or image_label", imageId)
	}
	meta.CredentialId, _ = obj.GetString("credential_id")
	if meta.CredentialId == "" {
		if registryId, _ := obj.GetString("registry_id"); registryId != "" {
			regObj, regErr := imagemodules.ContainerRegistries.Get(s, registryId, nil)
			if regErr != nil {
				return nil, httperrors.NewGeneralError(errors.Wrapf(regErr, "get container_registry %s for credential fallback", registryId))
			}
			meta.CredentialId, _ = regObj.GetString("credential_id")
		}
	}
	return meta, nil
}

func (man *SLLMImageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMImageCreateInput) (*api.LLMImageCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if nil != err {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	wantContainer := len(input.ContainerImageId) > 0 || input.ImageSource == api.LLM_IMAGE_SOURCE_CONTAINER_IMAGE
	if wantContainer {
		if len(input.ContainerImageId) == 0 {
			return input, httperrors.NewMissingParameterError("container_image_id")
		}
		meta, err := fetchContainerImageMeta(ctx, userCred, input.ContainerImageId)
		if err != nil {
			return input, err
		}
		input.ImageSource = api.LLM_IMAGE_SOURCE_CONTAINER_IMAGE
		input.ContainerImageId = meta.Id
		input.ImageName = meta.ImageName
		input.ImageLabel = meta.ImageLabel
		// credential is resolved at runtime from container_image
		input.CredentialId = ""
	} else {
		input.ImageSource = api.LLM_IMAGE_SOURCE_MANUAL
		input.ContainerImageId = ""
		if len(input.ImageName) == 0 {
			return input, httperrors.NewMissingParameterError("image_name")
		}
		if len(input.ImageLabel) == 0 {
			return input, httperrors.NewMissingParameterError("image_label")
		}
		if len(input.CredentialId) > 0 {
			cred, err := fetchImageCredential(ctx, userCred, input.CredentialId)
			if err != nil {
				return input, errors.Wrap(err, "fetchImageCredential")
			}
			input.CredentialId = cred.Id
		}
	}

	if len(input.LLMType) > 0 {
		if !api.IsLLMImageType(input.LLMType) {
			return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_IMAGE_TYPES.List(), ","))
		}
	}

	if input.LLMType == string(api.LLM_IMAGE_TYPE_DESKTOP) {
		appName, err := api.ResolveAppName(input.ImageName, input.ImageLabel, input.AppName)
		if err != nil {
			return input, errors.Wrap(httperrors.ErrInputParameter, err.Error())
		}
		input.AppName = appName
		cfg, err := api.ResolveDesktopConfig(input.ImageName, input.ImageLabel, input.DesktopConfig)
		if err != nil {
			return input, errors.Wrap(httperrors.ErrInputParameter, err.Error())
		}
		input.DesktopConfig = cfg
	} else {
		input.AppName = ""
		input.DesktopConfig = nil
	}

	input.Status = api.STATUS_READY
	return input, nil
}

func (man *SLLMImageManager) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMImageUpdateInput) (*api.LLMImageUpdateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if nil != err {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	if nil != input.CredentialId && len(*input.CredentialId) > 0 {
		cred, err := fetchImageCredential(ctx, userCred, *input.CredentialId)
		if err != nil {
			return input, errors.Wrap(err, "fetchImageCredential")
		}
		input.CredentialId = &cred.Id
	}

	if nil != input.LLMType && len(*input.LLMType) > 0 {
		if !api.IsLLMImageType(*input.LLMType) {
			return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_IMAGE_TYPES.List(), ","))
		}
	}

	return input, nil
}

func (image *SLLMImage) applyContainerImageLink(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMImageUpdateInput) error {
	wantContainer := false
	containerId := image.ContainerImageId

	switch {
	case input.ContainerImageId != nil:
		containerId = strings.TrimSpace(*input.ContainerImageId)
		wantContainer = containerId != ""
	case input.ImageSource != nil && *input.ImageSource == api.LLM_IMAGE_SOURCE_MANUAL:
		wantContainer = false
		containerId = ""
	case input.ImageSource != nil && *input.ImageSource == api.LLM_IMAGE_SOURCE_CONTAINER_IMAGE:
		wantContainer = true
		if containerId == "" {
			return httperrors.NewMissingParameterError("container_image_id")
		}
	default:
		wantContainer = image.ContainerImageId != "" || image.ImageSource == api.LLM_IMAGE_SOURCE_CONTAINER_IMAGE
	}

	if wantContainer {
		if containerId == "" {
			return httperrors.NewMissingParameterError("container_image_id")
		}
		meta, err := fetchContainerImageMeta(ctx, userCred, containerId)
		if err != nil {
			return err
		}
		src := api.LLM_IMAGE_SOURCE_CONTAINER_IMAGE
		empty := ""
		imageName := meta.ImageName
		imageLabel := meta.ImageLabel
		input.ImageSource = &src
		input.ContainerImageId = &meta.Id
		input.ImageName = &imageName
		input.ImageLabel = &imageLabel
		input.CredentialId = &empty
		return nil
	}

	src := api.LLM_IMAGE_SOURCE_MANUAL
	empty := ""
	input.ImageSource = &src
	input.ContainerImageId = &empty

	imageName := image.ImageName
	if input.ImageName != nil {
		imageName = *input.ImageName
	}
	imageLabel := image.ImageLabel
	if input.ImageLabel != nil {
		imageLabel = *input.ImageLabel
	}
	if imageName == "" {
		return httperrors.NewMissingParameterError("image_name")
	}
	if imageLabel == "" {
		return httperrors.NewMissingParameterError("image_label")
	}
	if input.CredentialId != nil && len(*input.CredentialId) > 0 {
		cred, err := fetchImageCredential(ctx, userCred, *input.CredentialId)
		if err != nil {
			return errors.Wrap(err, "fetchImageCredential")
		}
		input.CredentialId = &cred.Id
	}
	return nil
}

func (image *SLLMImage) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMImageUpdateInput) (api.LLMImageUpdateInput, error) {
	if err := image.applyContainerImageLink(ctx, userCred, &input); err != nil {
		return input, err
	}

	if input.LLMType != nil && len(*input.LLMType) > 0 {
		if !api.IsLLMImageType(*input.LLMType) {
			return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_IMAGE_TYPES.List(), ","))
		}
	}

	llmType := image.LLMType
	if input.LLMType != nil {
		llmType = *input.LLMType
	}
	imageName := image.ImageName
	if input.ImageName != nil {
		imageName = *input.ImageName
	}
	imageLabel := image.ImageLabel
	if input.ImageLabel != nil {
		imageLabel = *input.ImageLabel
	}

	if llmType == string(api.LLM_IMAGE_TYPE_DESKTOP) {
		explicitAppName := ""
		if input.AppName != nil {
			explicitAppName = *input.AppName
		}
		appName, err := api.ResolveAppName(imageName, imageLabel, explicitAppName)
		if err != nil {
			return input, errors.Wrap(httperrors.ErrInputParameter, err.Error())
		}
		input.AppName = &appName
		cfg, err := api.ResolveDesktopConfig(imageName, imageLabel, input.DesktopConfig)
		if err != nil {
			return input, errors.Wrap(httperrors.ErrInputParameter, err.Error())
		}
		input.DesktopConfig = cfg
	} else {
		empty := ""
		input.AppName = &empty
		input.DesktopConfig = nil
	}

	return input, nil
}

// GetResolvedDesktopConfig returns merged desktop config with defaults applied.
func (image *SLLMImage) GetResolvedDesktopConfig() (*api.LLMImageDesktopConfig, error) {
	if image.LLMType != string(api.LLM_IMAGE_TYPE_DESKTOP) {
		return nil, nil
	}
	return api.ResolveDesktopConfig(image.ImageName, image.ImageLabel, image.DesktopConfig)
}

func (man *SLLMImageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LLMImageListInput,
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
		q = q.Contains("image_name", input.ImageName)
	}
	if len(input.LLMType) > 0 {
		q = q.Equals("llm_type", input.LLMType)
	}
	if len(input.AppName) > 0 {
		q = q.Equals("app_name", input.AppName)
	}
	if len(input.ImageSource) > 0 {
		q = q.Equals("image_source", input.ImageSource)
	}
	if len(input.ContainerImageId) > 0 {
		q = q.Equals("container_image_id", input.ContainerImageId)
	}
	return q, nil
}

func (man *SLLMImageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LLMImageDetails {
	rows := make([]api.LLMImageDetails, len(objs))
	virtRows := man.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	images := make([]SLLMImage, 0)
	jsonutils.Update(&images, objs)
	containerImageIds := stringutils2.NewSortedStrings(nil)
	for i := range images {
		rows[i].SharableVirtualResourceDetails = virtRows[i]
		if images[i].ContainerImageId != "" {
			containerImageIds = containerImageIds.Append(images[i].ContainerImageId)
		}
	}
	if len(containerImageIds) == 0 {
		return rows
	}

	nameMap := map[string]string{}
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	for _, id := range containerImageIds {
		obj, err := imagemodules.ContainerImages.Get(s, id, nil)
		if err != nil {
			log.Warningf("FetchCustomizeColumns get container_image %s: %v", id, err)
			continue
		}
		name, _ := obj.GetString("name")
		nameMap[id] = name
	}
	for i := range images {
		if images[i].ContainerImageId != "" {
			rows[i].ContainerImage = nameMap[images[i].ContainerImageId]
		}
	}
	return rows
}

func (image *SLLMImage) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	for _, field := range []string{"llm_image_id"} {
		count, err := GetLLMManager().Query().Equals(field, image.Id).CountWithError()
		if err != nil {
			return errors.Wrap(err, "fetch llms")
		}
		if count > 0 {
			return errors.Wrapf(errors.ErrNotSupported, "This image is currently in use by %s in llms", field)
		}
		count, err = GetLLMSkuManager().Query().Equals("llm_image_id", image.Id).CountWithError()
		if err != nil {
			return errors.Wrap(err, "fetch llm models")
		}
		if count > 0 {
			return errors.Wrapf(errors.ErrNotSupported, "This image is currently in use by %s in llm models", field)
		}
	}
	return nil
}

// ApplyToContainerSpec fills image fields on a container spec.
// When linked to a container_image, only ContainerImageId is set; compute resolves Image/credential.
// Manual images set Image and ImageCredentialId from local columns.
func (image *SLLMImage) ApplyToContainerSpec(spec *commonapi.ContainerSpec) {
	if spec == nil || image == nil {
		return
	}
	if image.ContainerImageId != "" {
		spec.ContainerImageId = image.ContainerImageId
		spec.Image = ""
		spec.ImageCredentialId = ""
		return
	}
	spec.ContainerImageId = ""
	spec.Image = fmt.Sprintf("%s:%s", image.ImageName, image.ImageLabel)
	spec.ImageCredentialId = image.CredentialId
}

// ToContainerImage returns cached image_name:image_label (manual / display fallback).
// Prefer ApplyToContainerSpec when assembling ContainerSpec so linked images pass container_image_id.
func (image *SLLMImage) ToContainerImage() string {
	return fmt.Sprintf("%s:%s", image.ImageName, image.ImageLabel)
}

// GetImageCredentialId returns the stored credential for manual images.
func (image *SLLMImage) GetImageCredentialId() string {
	return image.CredentialId
}

// SyncContainerImageMeta refreshes cached image_name/image_label from linked container images.
// Source image deleted: keep cache and log warning (do not unlink).
func (man *SLLMImageManager) SyncContainerImageMeta(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	q := man.Query().IsNotEmpty("container_image_id")
	images := make([]SLLMImage, 0)
	err := db.FetchModelObjects(man, q, &images)
	if err != nil {
		log.Errorf("SyncContainerImageMeta FetchModelObjects: %v", err)
		return
	}
	for i := range images {
		img := &images[i]
		meta, err := fetchContainerImageMeta(ctx, nil, img.ContainerImageId)
		if err != nil {
			log.Warningf("SyncContainerImageMeta llm_image %s (%s) container_image %s: %v; keep cached %s:%s",
				img.Id, img.Name, img.ContainerImageId, err, img.ImageName, img.ImageLabel)
			continue
		}
		if img.ImageName == meta.ImageName && img.ImageLabel == meta.ImageLabel {
			continue
		}
		_, err = db.Update(img, func() error {
			img.ImageName = meta.ImageName
			img.ImageLabel = meta.ImageLabel
			if img.ImageSource != api.LLM_IMAGE_SOURCE_CONTAINER_IMAGE {
				img.ImageSource = api.LLM_IMAGE_SOURCE_CONTAINER_IMAGE
			}
			return nil
		})
		if err != nil {
			log.Errorf("SyncContainerImageMeta update llm_image %s: %v", img.Id, err)
			continue
		}
		log.Infof("SyncContainerImageMeta llm_image %s synced to %s:%s", img.Id, meta.ImageName, meta.ImageLabel)
	}
}
