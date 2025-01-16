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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/pod"
)

func init() {
	RegisterDriver(newExec())
}

type execDriver struct{}

func newExec() ILifecycle {
	return &execDriver{}
}

func (e execDriver) GetType() apis.ContainerLifecyleHandlerType {
	return apis.ContainerLifecyleHandlerTypeExec
}

func (e execDriver) Run(ctx context.Context, input *apis.ContainerLifecyleHandler, cri pod.CRI, id string) error {
	cfg := input.Exec
	resp, err := cri.ExecSync(ctx, id, cfg.Command, 0)
	if err != nil {
		return errors.Wrapf(err, "exec sync")
	}
	log.Infof("run command %v: stdout: %s, stderr: %s", cfg.Command, resp.Stdout, resp.Stderr)
	return nil
}
