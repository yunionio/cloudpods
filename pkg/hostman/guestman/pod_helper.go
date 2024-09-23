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

package guestman

import (
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/pod/image"
	"yunion.io/x/onecloud/pkg/util/pod/nerdctl"
)

func GetContainerdConnectionInfo() (string, string) {
	addr := options.HostOptions.ContainerRuntimeEndpoint
	addr = strings.TrimPrefix(addr, "unix://")
	namespace := "k8s.io"
	return addr, namespace
}

func NewContainerdImageTool() image.ImageTool {
	addr, namespace := GetContainerdConnectionInfo()
	return image.NewImageTool(addr, namespace)
}

func NewContainerdNerdctl() nerdctl.Nerdctl {
	addr, namespace := GetContainerdConnectionInfo()
	return nerdctl.NewNerdctl(addr, namespace)
}

func PullContainerdImage(input *hostapi.ContainerPullImageInput) error {
	opt := &image.PullOptions{
		RepoCommonOptions: image.RepoCommonOptions{
			SkipVerify: true,
		},
	}
	if input.Auth != nil {
		opt.Username = input.Auth.Username
		opt.Password = input.Auth.Password
	}
	imgTool := NewContainerdImageTool()
	output, err := imgTool.Pull(input.Image, opt)
	errs := make([]error, 0)
	if err != nil {
		// try http protocol
		errs = append(errs, errors.Wrapf(err, "pullImageByCtrCmd: %s", output))
		opt.PlainHttp = true
		log.Infof("try pull image %s by http", input.Image)
		if output2, err := imgTool.Pull(input.Image, opt); err != nil {
			errs = append(errs, errors.Wrapf(err, "pullImageByCtrCmd by http: %s", output2))
			return errors.NewAggregate(errs)
		}
	}
	return nil
}

func PushContainerdImage(input *hostapi.ContainerPushImageInput) error {
	opt := &image.PushOptions{
		RepoCommonOptions: image.RepoCommonOptions{
			SkipVerify: true,
		},
	}
	if input.Auth != nil {
		opt.Username = input.Auth.Username
		opt.Password = input.Auth.Password
	}
	imgTool := NewContainerdImageTool()
	err := imgTool.Push(input.Image, opt)
	errs := make([]error, 0)
	if err != nil {
		// try http protocol
		errs = append(errs, errors.Wrap(err, "pushImageByCtrCmd: %s"))
		opt.PlainHttp = true
		log.Infof("try push image %s by http", input.Image)
		if err := imgTool.Push(input.Image, opt); err != nil {
			errs = append(errs, errors.Wrapf(err, "pushImageByCtrCmd by http"))
			return errors.NewAggregate(errs)
		}
	}
	return nil
}
