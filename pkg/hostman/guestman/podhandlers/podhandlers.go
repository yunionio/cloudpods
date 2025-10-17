// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package podhandlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/util/proxy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/flushwriter"
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

func containerSyncActionHandler(cf containerActionFunc) appsrv.FilterHandler {
	return _containerActionHandler(cf, true, nil)
}

func containerActionHandler(cf containerActionFunc) appsrv.FilterHandler {
	return _containerActionHandler(cf, false, nil)
}

func containerActionHandlerWithWorker(cf containerActionFunc, workerMan *appsrv.SWorkerManager) appsrv.FilterHandler {
	return _containerActionHandler(cf, false, workerMan)
}

func _containerActionHandler(cf containerActionFunc, isSync bool, workerMan *appsrv.SWorkerManager) appsrv.FilterHandler {
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
		if isSync {
			data, err := cf(ctx, userCred, delayParams.pod, delayParams.containerId, delayParams.body)
			if err != nil {
				hostutils.Response(ctx, w, httperrors.NewBadRequestError("error: %v", err))
				return
			}
			hostutils.Response(ctx, w, data)
			return
		} else {
			delayFunc := func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
				dp := params.(*containerDelayActionParams)
				return cf(ctx, userCred, dp.pod, dp.containerId, dp.body)
			}
			if workerMan != nil {
				hostutils.DelayTaskWithWorker(ctx, delayFunc, delayParams, workerMan)
			} else {
				hostutils.DelayTask(ctx, delayFunc, delayParams)
			}
			hostutils.ResponseOk(ctx, w)
		}
	})
}

func AddPodHandlers(prefix string, app *appsrv.Application) {
	ctrHandlers := map[string]containerActionFunc{
		"create":                           createContainer,
		"delete":                           deleteContainer,
		"sync-status":                      syncContainerStatus,
		"pull-image":                       pullImage,
		"save-volume-mount-to-image":       saveVolumeMountToImage,
		"commit":                           commitContainer,
		"add-volume-mount-post-overlay":    containerAddVolumeMountPostOverlay,
		"remove-volume-mount-post-overlay": containerRemoveVolumeMountPostOverlay,
	}
	for action, f := range ctrHandlers {
		app.AddHandler("POST",
			fmt.Sprintf("%s/pods/%s/containers/%s/%s", prefix, POD_ID, CONTAINER_ID, action),
			containerActionHandler(f))
	}

	startWorker := appsrv.NewWorkerManager("container-start-worker", options.HostOptions.ContainerStartWorkerCount, appsrv.DEFAULT_BACKLOG, false)
	stopWorker := appsrv.NewWorkerManager("container-stop-worker", options.HostOptions.ContainerStopWorkerCount, appsrv.DEFAULT_BACKLOG, false)

	ctrWorkerHanders := map[string]struct {
		workerMan *appsrv.SWorkerManager
		f         containerActionFunc
	}{
		"start": {startWorker, startContainer},
		"stop":  {stopWorker, stopContainer},
	}
	for action, fw := range ctrWorkerHanders {
		app.AddHandler("POST",
			fmt.Sprintf("%s/pods/%s/containers/%s/%s", prefix, POD_ID, CONTAINER_ID, action),
			containerActionHandlerWithWorker(fw.f, fw.workerMan))
	}

	execWorker := appsrv.NewWorkerManager("container-exec-worker", 16, appsrv.DEFAULT_BACKLOG, false)
	app.AddHandler3(newContainerWorkerHandler("POST", fmt.Sprintf("%s/pods/%s/containers/%s/exec-sync", prefix, POD_ID, CONTAINER_ID), execWorker, containerSyncActionHandler(containerExecSync)))
	app.AddHandler3(newContainerWorkerHandler("POST", fmt.Sprintf("%s/pods/%s/containers/%s/exec", prefix, POD_ID, CONTAINER_ID), execWorker, execContainer()))

	logWorker := appsrv.NewWorkerManager("container-log-worker", 64, appsrv.DEFAULT_BACKLOG, false)
	app.AddHandler3(newContainerWorkerHandler("GET", fmt.Sprintf("%s/pods/%s/containers/%s/log", prefix, POD_ID, CONTAINER_ID), logWorker, logContainer()))

	syncWorker := appsrv.NewWorkerManager("container-sync-action-worker", 16, appsrv.DEFAULT_BACKLOG, false)
	app.AddHandler3(newContainerWorkerHandler("POST", fmt.Sprintf("%s/pods/%s/containers/%s/set-resources-limit", prefix, POD_ID, CONTAINER_ID), syncWorker, containerSyncActionHandler(containerSetResourcesLimit)))
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
	input := new(hostapi.ContainerStopInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrapf(err, "unmarshal to ContainerStopInput: %s", body.String())
	}
	return pod.StopContainer(ctx, userCred, ctrId, input)
}

func deleteContainer(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return pod.DeleteContainer(ctx, userCred, containerId)
}

func syncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return pod.SyncContainerStatus(ctx, userCred, id)
}

func saveVolumeMountToImage(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(hostapi.ContainerSaveVolumeMountToImageInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to input")
	}
	return pod.SaveVolumeMountToImage(ctx, userCred, input, ctrId)
}

func newContainerWorkerHandler(method, urlPath string, worker *appsrv.SWorkerManager, hander appsrv.FilterHandler) *appsrv.SHandlerInfo {
	hi := &appsrv.SHandlerInfo{}
	hi.SetMethod(method)
	hi.SetPath(urlPath)
	hi.SetHandler(hander)
	hi.SetProcessTimeout(1 * time.Hour)
	hi.SetWorkerManager(worker)
	return hi
}

type containerWorkerActionHander func(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, ctrId string, query, body jsonutils.JSONObject, r *http.Request, w http.ResponseWriter)

func containerWorkerAction(handler containerWorkerActionHander) appsrv.FilterHandler {
	return auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		params, query, body := appsrv.FetchEnv(ctx, w, r)
		podId := params[POD_ID]
		ctrId := params[CONTAINER_ID]
		userCred := auth.FetchUserCredential(ctx, nil)
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
		handler(ctx, userCred, pod, ctrId, query, body, r, w)
	})
}

func logContainer() appsrv.FilterHandler {
	return containerWorkerAction(func(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, ctrId string, query, body jsonutils.JSONObject, r *http.Request, w http.ResponseWriter) {
		input := new(compute.PodLogOptions)
		if err := query.Unmarshal(input); err != nil {
			hostutils.Response(ctx, w, errors.Wrap(err, "unmarshal to PodLogOptions"))
			return
		}
		if err := compute.ValidatePodLogOptions(input); err != nil {
			hostutils.Response(ctx, w, err)
			return
		}
		if _, ok := w.(http.Flusher); !ok {
			hostutils.Response(ctx, w, errors.Errorf("unable to convert to http.Flusher"))
			return
		}
		w.Header().Set("Transfer-Encoding", "chunked")
		fw := flushwriter.Wrap(w)
		ctx, cancel := context.WithCancel(ctx)
		go func() {
			for {
				// check whether client request is closed
				select {
				case <-r.Context().Done():
					log.Infof("client request is closed, end session")
					cancel()
					return
				}
			}
		}()
		if err := pod.ReadLogs(ctx, userCred, ctrId, input, fw, fw); err != nil {
			hostutils.Response(ctx, w, errors.Wrap(err, "Read logs"))
			return
		}
	})
}

func execContainer() appsrv.FilterHandler {
	return containerWorkerAction(func(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, ctrId string, query, body jsonutils.JSONObject, r *http.Request, w http.ResponseWriter) {
		input := new(compute.ContainerExecInput)
		if err := query.Unmarshal(input); err != nil {
			hostutils.Response(ctx, w, errors.Wrap(err, "unmarshal to ContainerExecInput"))
			return
		}
		criUrl, err := pod.ExecContainer(ctx, userCred, ctrId, input)
		if err != nil {
			hostutils.Response(ctx, w, errors.Wrap(err, "get exec url"))
			return
		}
		proxyStream(w, r, criUrl)
		return
	})
}

type responder struct {
	errorMessage string
}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func proxyStream(w http.ResponseWriter, r *http.Request, url *url.URL) {
	handler := proxy.NewUpgradeAwareHandler(url, nil, false, true, &responder{})
	handler.ServeHTTP(w, r)
}

func containerExecSync(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(compute.ContainerExecSyncInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerExecSyncInput")
	}
	return pod.ContainerExecSync(ctx, userCred, containerId, input)
}

func containerSetResourcesLimit(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(apis.ContainerResources)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerExecSyncInput")
	}
	return pod.SetContainerResourceLimit(containerId, input)
}

func commitContainer(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(hostapi.ContainerCommitInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerCommitInput")
	}
	return pod.CommitContainer(ctx, userCred, ctrId, input)
}

func containerAddVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(compute.ContainerVolumeMountAddPostOverlayInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerVolumeMountAddPostOverlayInput")
	}
	return nil, pod.AddContainerVolumeMountPostOverlay(ctx, userCred, containerId, input)
}

func containerRemoveVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, pod guestman.PodInstance, containerId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(compute.ContainerVolumeMountRemovePostOverlayInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to ContainerMountVolumeRemovePostOverlayInput")
	}
	return nil, pod.RemoveContainerVolumeMountPostOverlay(ctx, userCred, containerId, input)
}
