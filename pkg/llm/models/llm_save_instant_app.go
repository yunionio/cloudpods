package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func (llm *SLLM) GetDetailsProbedPackages(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	pkgInfos, err := llm.getProbedPackagesExt(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "getProbedPackagesExt")
	}
	return jsonutils.Marshal(pkgInfos), nil
}

func (llm *SLLM) getProbedPackagesExt(ctx context.Context, userCred mcclient.TokenCredential, pkgAppIds ...string) (map[string]api.LLMInternalPkgInfo, error) {
	drv := llm.GetLLMContainerDriver()
	return drv.GetProbedPackagesExt(ctx, userCred, llm, pkgAppIds...)
}
