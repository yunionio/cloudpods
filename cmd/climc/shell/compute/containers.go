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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/ghodss/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/pod/stream/cp"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Containers)
	cmd.Create(new(options.ContainerCreateOptions))
	cmd.List(new(options.ContainerListOptions))
	cmd.Show(new(options.ContainerShowOptions))
	cmd.GetMetadata(new(options.ServerIdOptions))
	cmd.BatchDelete(new(options.ContainerDeleteOptions))
	cmd.BatchPerform("stop", new(options.ContainerStopOptions))
	cmd.BatchPerform("start", new(options.ContainerStartOptions))
	cmd.BatchPerform("syncstatus", new(options.ContainerIdsOptions))
	cmd.Perform("save-volume-mount-image", new(options.ContainerSaveVolumeMountImage))
	cmd.Perform("exec-sync", new(options.ContainerExecSyncOptions))
	cmd.BatchPerform("set-resources-limit", new(options.ContainerSetResourcesLimitOptions))
	cmd.Perform("commit", new(options.ContainerCommitOptions))
	cmd.Perform("add-volume-mount-post-overlay", new(options.ContainerAddVolumeMountPostOverlayOptions))
	cmd.Perform("remove-volume-mount-post-overlay", new(options.ContainerRemoveVolumeMountPostOverlayOptions))

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
		if err := man.LogToWriter(s, opts.ID, input, os.Stdout); err != nil {
			return errors.Wrap(err, "get container log")
		}
		return nil
	})

	R(new(options.ContainerCopyOptions), "container-cp", "Container copy", func(s *mcclient.ClientSession, opts *options.ContainerCopyOptions) error {
		parts := strings.Split(opts.CONTAINER_ID_FILE, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid container id: %s", opts.CONTAINER_ID_FILE)
		}
		if opts.RawFile {
			fr, err := os.Open(opts.SRC_FILE)
			if err != nil {
				return errors.Wrapf(err, "open file: %v", opts.SRC_FILE)
			}
			defer fr.Close()
			if err := modules.Containers.CopyTo(s, parts[0], parts[1], fr); err != nil {
				return errors.Wrapf(err, "copy file to container")
			}
			return nil
		} else {
			return cp.NewCopy().CopyToContainer(s, opts.SRC_FILE, cp.ContainerFileOpt{
				ContainerId: parts[0],
				File:        parts[1],
			})
		}
	})

	R(new(options.ContainerCopyOptions), "container-cp-from", "Container copy", func(s *mcclient.ClientSession, opts *options.ContainerCopyOptions) error {
		parts := strings.Split(opts.SRC_FILE, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid container id: %s", opts.CONTAINER_ID_FILE)
		}
		ctrId := parts[0]
		ctrFile := parts[1]
		destFile := opts.CONTAINER_ID_FILE
		if opts.RawFile {
			fw, err := os.Create(destFile)
			if err != nil {
				return errors.Wrapf(err, "open file: %v", destFile)
			}
			defer fw.Close()
			if err := modules.Containers.CopyFrom(s, ctrId, ctrFile, fw); err != nil {
				return errors.Wrap(err, "copy from")
			}
			return nil
		} else {
			return cp.NewCopy().CopyFromContainer(s, cp.ContainerFileOpt{
				ContainerId: ctrId,
				File:        ctrFile,
			}, destFile)
		}
	})
}
