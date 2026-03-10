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

	// generate post overlay info
	{
		err = llm.UpdateMountedModelFullNames(ctx, userCred, nil, true, input.LLMImageId, input.LLMSkuId)
		if err != nil {
			return nil, errors.Wrap(err, "UpdateMountedModelFullNames")
		}
	}

	lcd := llm.GetLLMContainerDriver()
	containers := GetDriverPodContainers(ctx, lcd, llm, llmImage, sku, nil, nil, "")

	data.Pod = &computeapi.PodCreateInput{
		HostIPC:    true,
		Containers: containers,
	}

	return data, nil
}
