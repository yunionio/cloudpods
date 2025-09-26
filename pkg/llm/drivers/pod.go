package drivers

import (
	"context"
	"time"

	"github.com/golang-plus/errors"
	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

// var PodDriver models.IPodDriver

// func init() {
// 	PodDriver = &SPodDriver{}
// }

type SPodDriver struct{}

func (p *SPodDriver) RequestCreatePodWithPolling(ctx context.Context, userCred mcclient.TokenCredential, input *computeapi.ServerCreateInput) (string, error) {
	session := auth.GetSession(ctx, userCred, "")
	server, err := modules.Servers.Create(session, jsonutils.Marshal(input))
	if err != nil {
		return "", errors.Wrap(err, "CreateServer")
	}

	serverId, err := server.GetString("id")
	if err != nil {
		return "", errors.Wrap(err, "Get server id")
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		detail, err := modules.Servers.Get(session, serverId, nil)
		if err != nil {
			return "", errors.Wrap(err, "Get server detail")
		}

		status, _ := detail.GetString("status")
		if status == "running" {
			break
		}

		time.Sleep(10 * time.Second)
	}

	return serverId, nil
}

func (p *SPodDriver) RequestCreatePod(ctx context.Context, userCred mcclient.TokenCredential, input *computeapi.ServerCreateInput) (jsonutils.JSONObject, error) {
	session := auth.GetSession(ctx, userCred, "")
	server, err := modules.Servers.Create(session, jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrap(err, "CreateServer")
	}

	return server, nil
}

func (p *SPodDriver) RequestExecSyncContainer(ctx context.Context, userCred mcclient.TokenCredential, containerId string, input *computeapi.ContainerExecSyncInput) (jsonutils.JSONObject, error) {
	session := auth.GetSession(ctx, userCred, "")
	output, err := modules.Containers.PerformAction(session, containerId, "exec-sync", jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrap(err, "ExecSync")
	}

	return output, nil
}

func (p *SPodDriver) RequestGetContainersByPodId(ctx context.Context, userCred mcclient.TokenCredential, podId string) ([]jsonutils.JSONObject, error) {
	session := auth.GetSession(ctx, userCred, "")
	params := jsonutils.NewDict()
	params.Set("guest_id", jsonutils.NewString(podId))
	output, err := modules.Containers.List(session, params)
	if err != nil {
		return nil, errors.Wrap(err, "ListContainers")
	}

	return output.Data, nil
}

func (p *SPodDriver) RequestCreateContainerOnPod(ctx context.Context, userCred mcclient.TokenCredential, podId string, input *computeapi.PodContainerCreateInput) (jsonutils.JSONObject, error) {
	session := auth.GetSession(ctx, userCred, "")
	params := &computeapi.ContainerCreateInput{
		GuestId:  podId,
		Spec:     input.ContainerSpec,
		SkipTask: true,
	}
	params.Name = input.Name
	return modules.Containers.Create(session, jsonutils.Marshal(params))
}

func (p *SPodDriver) RequestDoCreateContainer(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string) error {
	params := jsonutils.NewDict()
	params.Set("auto_start", jsonutils.JSONTrue)
	_, err := requesyContainerHostActionWithTask(ctx, userCred, containerId, "", "ContainerCreateTask", taskId, params)

	return err
}

func (p *SPodDriver) RequestDownloadFileIntoContainer(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string, input *computeapi.ContainerDownloadFileInput) (jsonutils.JSONObject, error) {
	return requesyContainerHostActionWithTask(ctx, userCred, containerId, "download-file", "", taskId, jsonutils.Marshal(input))
}

func (p *SPodDriver) RequestOllamaBlobsCache(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string, input *api.LLMAccessCacheInput) (jsonutils.JSONObject, error) {
	return requesyContainerHostActionWithTask(ctx, userCred, containerId, "access-ollama-blobs-cache", "", taskId, jsonutils.Marshal(input))
}

func requesyContainerHostActionWithTask(ctx context.Context, userCred mcclient.TokenCredential, containerId string, hostAction string, containerTask string, taskId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	session := auth.GetSession(ctx, userCred, "")
	input := &computeapi.ContainerRequestHostActionByOtherServiceInput{
		HostAction:    hostAction,
		ContainerTask: containerTask,
		TaskId:        taskId,
		Body:          body,
	}
	return modules.Containers.PerformAction(session, containerId, "request-host-action-by-other-service", jsonutils.Marshal(input))
}
