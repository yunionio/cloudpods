package models

import (
	"context"

	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func GetLLMPodCreateInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMCreateInput,
	llm *SLLM,
	sku *SLLMSku,
	llmImage *SLLMImage,
	eip string,
) (*computeapi.ServerCreateInput, error) {
	data, err := GetLLMBasePodCreateInput(ctx, userCred, &input.LLMBaseCreateInput, &llm.SLLMBase, &sku.SLLMSkuBase, eip)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMBasePodCreateInput: ")
	}

	lcd := llm.GetLLMContainerDriver()
	llmContainer := lcd.GetContainerSpec(ctx, llm, llmImage, sku, nil, nil, "")

	data.Pod = &computeapi.PodCreateInput{
		HostIPC: true,
		Containers: []*computeapi.PodContainerCreateInput{
			llmContainer,
		},
	}

	return data, nil
}
