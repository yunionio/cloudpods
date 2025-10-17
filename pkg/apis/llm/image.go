package llm

import "yunion.io/x/onecloud/pkg/apis"

type LLMImageListInput struct {
	apis.SharableVirtualResourceListInput

	ImageLabel string `json:"image_label"`
	ImageName  string `json:"image_name"`
}

type LLMImageCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName    string `json:"image_name"`
	ImageLabel   string `json:"image_label"`
	CredentialId string `json:"credential_id"`
}

type LLMImageUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName    *string `json:"image_name,omitempty"`
	ImageLabel   *string `json:"image_label,omitempty"`
	CredentialId *string `json:"credential_id,omitempty"`
}
