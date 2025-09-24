package podhandlers

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"
)

type llmActionFunc func(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, llmId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error)

type llmDelayActionParams struct {
	pod         guestman.PodInstance
	containerId string
	llmId       string
	body        jsonutils.JSONObject
}

func llmActionHandler(cf llmActionFunc) appsrv.FilterHandler {
	return auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		params, _, body := appsrv.FetchEnv(ctx, w, r)
		podId := params[POD_ID]
		ctrId := params[CONTAINER_ID]
		// llmId := params[LLM_ID]
		userCred := auth.FetchUserCredential(ctx, nil)
		if body == nil {
			body = jsonutils.NewDict()
		}
		podObj, ok := guestman.GetGuestManager().GetServer(podId)
		if !ok {
			hostutils.Response(ctx, w, httperrors.NewNotFoundError("Not found pod %s", podId))
			return
		}
		pod, ok := podObj.(guestman.PodInstance)
		if !ok {
			hostutils.Response(ctx, w, httperrors.NewBadRequestError("runtime instance is %#v", podObj))
			return
		}
		delayParams := &llmDelayActionParams{
			pod:         pod,
			containerId: ctrId,
			llmId:       "",
			body:        body,
		}
		delayFunc := func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
			dp := params.(*llmDelayActionParams)
			return cf(ctx, userCred, dp.pod, dp.containerId, dp.llmId, dp.body)
		}
		hostutils.DelayTask(ctx, delayFunc, delayParams)
		hostutils.ResponseOk(ctx, w)
	})
}

func accessModelCacheHandler(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, llmId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(llm.LLMAccessCacheInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, err
	}
	cacheManager := storageman.GetManager().LocalStorageImagecacheManager.(*storageman.SLocalImageCacheManager)
	if cacheManager == nil {
		return nil, errors.Error("Can't get LocalStorageImagecacheManager")
	}
	for _, blob := range input.Blobs {
		if err := cacheManager.AccessModelCache(ctx, input.ModelName, blob); nil != err {
			return nil, errors.Wrapf(err, "Failed to attatch model cache")
		}
	}
	return jsonutils.NewDict(), nil
}
