package podhandlers

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	POD_ID       = "<podId>"
	CONTAINER_ID = "<containerId>"
)

type containerActionFunc func(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error)

type containerDelayActionParams struct {
	pod         guestman.PodInstance
	containerId string
	body        jsonutils.JSONObject
}

func containerActionHandler(cf containerActionFunc) appsrv.FilterHandler {
	return auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		params, _, body := appsrv.FetchEnv(ctx, w, r)
		podId := params[POD_ID]
		ctrId := params[CONTAINER_ID]
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
		delayParams := &containerDelayActionParams{
			pod:         pod,
			containerId: ctrId,
			body:        body,
		}
		hostutils.DelayTask(ctx, func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
			dp := params.(*containerDelayActionParams)
			return cf(ctx, userCred, dp.pod, dp.containerId, dp.body)
		}, delayParams)
		hostutils.ResponseOk(ctx, w)
	})
}

func AddPodHandlers(prefix string, app *appsrv.Application) {
	ctrHandlers := map[string]containerActionFunc{
		"create":      createContainer,
		"start":       startContainer,
		"stop":        stopContainer,
		"delete":      deleteContainer,
		"sync-status": syncContainerStatus,
		"pull-image":  pullImage,
	}
	for action, f := range ctrHandlers {
		app.AddHandler("POST",
			fmt.Sprintf("%s/pods/%s/containers/%s/%s", prefix, POD_ID, CONTAINER_ID, action),
			containerActionHandler(f))
	}
}

func pullImage(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(hostapi.ContainerPullImageInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerPullImageInput")
	}
	return pod.PullImage(ctx, userCred, ctrId, input)
}

func createContainer(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(hostapi.ContainerCreateInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerCreateInput")
	}
	return pod.CreateContainer(ctx, userCred, id, input)
}

func startContainer(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(hostapi.ContainerCreateInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerCreateInput")
	}
	return pod.StartContainer(ctx, userCred, containerId, input)
}

func stopContainer(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return pod.StopContainer(ctx, userCred, ctrId, body)
}

func deleteContainer(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return pod.DeleteContainer(ctx, userCred, containerId)
}

func syncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return pod.SyncContainerStatus(ctx, userCred, id)
}
