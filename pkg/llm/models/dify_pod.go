package models

import (
	"context"

	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func GetDifyPodCreateInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.DifyCreateInput,
	dify *SDify,
	sku *SDifyModel,
	eip string,
) (*computeapi.ServerCreateInput, error) {
	data, err := GetLLMBasePodCreateInput(ctx, userCred, &input.LLMBaseCreateInput, &dify.SLLMBase, &sku.SLLMModelBase, eip)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMBasePodCreateInput: ")
	}

	ctrs := dify.GetDifyContainers()

	data.Pod = &computeapi.PodCreateInput{
		HostIPC:    true,
		Containers: ctrs,
	}

	return data, nil
}
