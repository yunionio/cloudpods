package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IContainerTask interface {
	taskman.ITask

	GetContainer() *SContainer
	GetPod() *SGuest
}

type IPodDriver interface {
	IGuestDriver

	RequestCreateContainer(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestStartContainer(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestStopContainer(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestDeleteContainer(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestSyncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestPullContainerImage(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestSaveVolumeMountImage(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestExecContainer(ctx context.Context, userCred mcclient.TokenCredential, ctr *SContainer, input *compute.ContainerExecInput) error
}
