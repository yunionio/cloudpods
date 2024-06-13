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

package compute

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/ghodss/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Containers)
	cmd.Create(new(options.ContainerCreateOptions))
	cmd.List(new(options.ContainerListOptions))
	cmd.Show(new(options.ContainerShowOptions))
	cmd.BatchDelete(new(options.ContainerDeleteOptions))
	cmd.BatchPerform("stop", new(options.ContainerStopOptions))
	cmd.BatchPerform("start", new(options.ContainerStartOptions))
	cmd.BatchPerform("syncstatus", new(options.ContainerIdsOptions))
	cmd.Perform("save-volume-mount-image", new(options.ContainerSaveVolumeMountImage))
	cmd.Perform("exec-sync", new(options.ContainerExecSyncOptions))

	type UpdateSpecOptions struct {
		ID string `help:"ID or name of server" json:"-"`
	}
	R(&UpdateSpecOptions{}, "container-update-spec", "Update spec of a container", func(s *mcclient.ClientSession, opts *UpdateSpecOptions) error {
		result, err := modules.Containers.Get(s, opts.ID, nil)
		if err != nil {
			return errors.Wrap(err, "get container id")
		}
		yamlBytes := result.YAMLString()
		tempfile, err := ioutil.TempFile("", fmt.Sprintf("container-%s*.yaml", opts.ID))
		if err != nil {
			return err
		}
		defer os.Remove(tempfile.Name())
		if _, err := tempfile.Write([]byte(yamlBytes)); err != nil {
			return err
		}
		if err := tempfile.Close(); err != nil {
			return err
		}

		cmd := exec.Command("vim", tempfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return err
		}
		content, err := ioutil.ReadFile(tempfile.Name())
		if err != nil {
			return err
		}
		jsonBytes, err := yaml.YAMLToJSON(content)
		if err != nil {
			return err
		}
		body, err := jsonutils.Parse(jsonBytes)
		if err != nil {
			return err
		}
		if _, err := modules.Containers.Update(s, opts.ID, body); err != nil {
			return errors.Wrap(err, "update spec")
		}

		return nil
	})

	type SetSpecOptions struct {
		ID                string `help:"ID or name of server" json:"-"`
		EnableSimulateCpu bool   `help:"Enable simulating /sys/devices/system/cpu directory"`
	}
	R(&SetSpecOptions{}, "container-set-spec", "Set spec of a container", func(s *mcclient.ClientSession, opts *SetSpecOptions) error {
		result, err := modules.Containers.Get(s, opts.ID, nil)
		if err != nil {
			return errors.Wrap(err, "get container id")
		}
		spec := new(computeapi.ContainerSpec)
		if err := result.Unmarshal(spec, "spec"); err != nil {
			return errors.Wrap(err, "unmarshal to spec")
		}
		spec.SimulateCpu = opts.EnableSimulateCpu
		result.(*jsonutils.JSONDict).Set("spec", jsonutils.Marshal(spec))

		if _, err := modules.Containers.Update(s, opts.ID, result); err != nil {
			return errors.Wrap(err, "update spec")
		}
		return nil
	})

	R(new(options.ContainerExecOptions), "container-exec", "Container exec", func(s *mcclient.ClientSession, opts *options.ContainerExecOptions) error {
		man := modules.Containers
		return man.Exec(s, opts.ID, opts.ToAPIInput())
	})

	R(new(options.ContainerLogOptions), "container-log", "Get container log", func(s *mcclient.ClientSession, opts *options.ContainerLogOptions) error {
		man := modules.Containers
		input, err := opts.ToAPIInput()
		if err != nil {
			return err
		}
		reader, err := man.Log(s, opts.ID, input)
		if err != nil {
			return errors.Wrap(err, "get container log")
		}
		defer reader.Close()

		r := bufio.NewReader(reader)
		for {
			bytes, err := r.ReadBytes('\n')
			if _, err := os.Stdout.Write(bytes); err != nil {
				return errors.Wrap(err, "write container log to stdout")
			}
			if err != nil {
				if err != io.EOF {
					return errors.Wrap(err, "read container log")
				}
				return nil
			}
		}
	})
}
