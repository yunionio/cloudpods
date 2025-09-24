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

package models

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
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
	RequestCommitContainer(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestSaveVolumeMountImage(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestExecSyncContainer(ctx context.Context, userCred mcclient.TokenCredential, ctr *SContainer, input *compute.ContainerExecSyncInput) (jsonutils.JSONObject, error)
	RequestSetContainerResourcesLimit(ctx context.Context, userCred mcclient.TokenCredential, c *SContainer, limit *apis.ContainerResources) (jsonutils.JSONObject, error)

	RequestAddVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
	RequestRemoveVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, task IContainerTask) error
}
