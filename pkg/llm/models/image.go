package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/pkg/errors"
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

	input.Status = api.LLM_STATUS_READY
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
	return input, nil
}

func (image *SLLMImage) ToContainerImage() string {
	return fmt.Sprintf("%s:%s", image.ImageName, image.ImageLabel)
}
