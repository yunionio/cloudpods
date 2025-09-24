package models

import (
	"context"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/printutils"
)

type IPodDriver interface {
	RequestCreatePod(ctx context.Context, userCred mcclient.TokenCredential, input *computeapi.ServerCreateInput) (jsonutils.JSONObject, error)

	RequestExecSyncContainer(ctx context.Context, userCred mcclient.TokenCredential, containerId string, input *computeapi.ContainerExecSyncInput) (jsonutils.JSONObject, error)
	RequestGetContainersByPodId(ctx context.Context, userCred mcclient.TokenCredential, podId string) (*printutils.ListResult, error)
	RequestCreateContainerOnPod(ctx context.Context, userCred mcclient.TokenCredential, podId string, input *computeapi.PodContainerCreateInput) (jsonutils.JSONObject, error)

	RequestDoCreateContainer(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string) error

	RequestDownloadFileIntoContainer(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string, input *computeapi.ContainerDownloadFileInput) (jsonutils.JSONObject, error)
	RequestOllamaBlobsCache(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string, input *api.LLMAccessCacheInput) (jsonutils.JSONObject, error)
}
