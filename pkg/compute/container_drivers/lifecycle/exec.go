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

package lifecycle

import (
	"context"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterContainerLifecyleDriver(newExec())
}

type execDriver struct{}

func newExec() models.IContainerLifecyleDriver {
	return &execDriver{}
}

func (e execDriver) GetType() apis.ContainerLifecyleHandlerType {
	return apis.ContainerLifecyleHandlerTypeExec
}

func (e execDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *apis.ContainerLifecyleHandler) error {
	if input.Exec == nil {
		return httperrors.NewNotEmptyError("exec field")
	}
	if len(input.Exec.Command) == 0 {
		return httperrors.NewNotEmptyError("command is required")
	}
	return nil
}
