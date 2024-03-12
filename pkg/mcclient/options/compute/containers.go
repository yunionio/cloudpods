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
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ContainerListOptions struct {
	options.BaseListOptions
	GuestId string `json:"guest_id" help:"guest(pod) id or name"`
}

func (o *ContainerListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type ContainerDeleteOptions struct {
	ServerIdsOptions
}

type ContainerCreateOptions struct {
	PODID       string   `help:"Name or id of server pod" json:"-"`
	NAME        string   `help:"Name of container" json:"-"`
	IMAGE       string   `help:"Image of container" json:"image"`
	Command     []string `help:"Command to execute (i.e., entrypoint for docker)" json:"command"`
	Args        []string `help:"Args for the Command (i.e. command for docker)" json:"args"`
	WorkingDir  string   `help:"Current working directory of the command" json:"working_dir"`
	Env         []string `help:"List of environment variable to set in the container and the format is: <key>=<value>"`
	VolumeMount []string `help:"Volume mount of the container and the format is: name=<val>,mount=<container_path>,readonly=<true_or_false>,disk_index=<disk_number>,disk_id=<disk_id>"`
}

func (o *ContainerCreateOptions) Params() (jsonutils.JSONObject, error) {
	req := computeapi.ContainerCreateInput{
		GuestId: o.PODID,
		Spec: computeapi.ContainerSpec{
			ContainerSpec: apis.ContainerSpec{
				Image:        o.IMAGE,
				Command:      o.Command,
				Args:         o.Args,
				WorkingDir:   o.WorkingDir,
				Envs:         make([]*apis.ContainerKeyValue, 0),
				VolumeMounts: make([]*apis.ContainerVolumeMount, 0),
			},
		},
	}
	req.Name = o.NAME
	for _, env := range o.Env {
		e, err := parseContainerEnv(env)
		if err != nil {
			return nil, errors.Wrapf(err, "parseContainerEnv %s", env)
		}
		req.Spec.Envs = append(req.Spec.Envs, e)
	}
	for _, vmStr := range o.VolumeMount {
		vm, err := parseContainerVolumeMount(vmStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parseContainerVolumeMount %s", vmStr)
		}
		req.Spec.VolumeMounts = append(req.Spec.VolumeMounts, vm)
	}
	return jsonutils.Marshal(req), nil
}

func parseContainerEnv(env string) (*apis.ContainerKeyValue, error) {
	kv := strings.Split(env, "=")
	if len(kv) != 2 {
		return nil, errors.Errorf("invalid env: %q", env)
	}
	return &apis.ContainerKeyValue{
		Key:   kv[0],
		Value: kv[1],
	}, nil
}

func parseContainerVolumeMount(vmStr string) (*apis.ContainerVolumeMount, error) {
	vm := &apis.ContainerVolumeMount{}
	for _, seg := range strings.Split(vmStr, ",") {
		info := strings.Split(seg, "=")
		if len(info) != 2 {
			return nil, errors.Errorf("invalid option %s", seg)
		}
		key := info[0]
		val := info[1]
		switch key {
		case "read_only", "ro", "readonly":
			if strings.ToLower(val) == "true" {
				vm.ReadOnly = true
			}
		case "mount_path":
			vm.MountPath = val
		case "host_path":
			if vm.HostPath == nil {
				vm.HostPath = &apis.ContainerVolumeMountHostPath{}
			}
			vm.Type = apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH
			vm.HostPath.Path = val
		case "disk_index":
			vm.Type = apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK
			if vm.Disk == nil {
				vm.Disk = &apis.ContainerVolumeMountDisk{}
			}
			index, err := strconv.Atoi(val)
			if err != nil {
				return nil, errors.Wrapf(err, "wrong disk_index %s", val)
			}
			vm.Disk.Index = &index
		case "disk_id":
			vm.Type = apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK
			if vm.Disk == nil {
				vm.Disk = &apis.ContainerVolumeMountDisk{}
			}
			vm.Disk.Id = val
		case "disk_subdir", "disk_sub_dir", "disk_sub_directory":
			vm.Type = apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK
			if vm.Disk == nil {
				vm.Disk = &apis.ContainerVolumeMountDisk{}
			}
			vm.Disk.SubDirectory = val
		case "disk_storage_size_file", "disk_ssf":
			vm.Type = apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK
			if vm.Disk == nil {
				vm.Disk = &apis.ContainerVolumeMountDisk{}
			}
			vm.Disk.StorageSizeFile = val
		}
	}
	return vm, nil
}

type ContainerStopOptions struct {
	ServerIdsOptions
	Timeout int `help:"Stopping timeout" json:"timeout"`
}

func (o *ContainerStopOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ContainerStartOptions struct {
	ServerIdsOptions
}
