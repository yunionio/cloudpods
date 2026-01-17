package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
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

	ImageName  string `width:"128" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"user"`
	ImageLabel string `width:"64" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"user"`

	CredentialId string `width:"128" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" update:"user"`
	LLMType      string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"user"`
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

func (man *SLLMImageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMImageCreateInput) (*api.LLMImageCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if nil != err {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	if len(input.CredentialId) > 0 {
		cred, err := fetchImageCredential(ctx, userCred, input.CredentialId)
		if err != nil {
			return input, errors.Wrap(err, "fetchImageCredential")
		}
		input.CredentialId = cred.Id
	}

	if len(input.LLMType) > 0 {
		if !api.IsLLMImageType(input.LLMType) {
			return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_IMAGE_TYPES.List(), ","))
		}
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
		q = q.Equals("image_name", input.ImageName)
	}
	if len(input.LLMType) > 0 {
		q = q.Equals("llm_type", input.LLMType)
	}
	return q, nil
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

func (image *SLLMImage) ToContainerImage() string {
	return fmt.Sprintf("%s:%s", image.ImageName, image.ImageLabel)
}
