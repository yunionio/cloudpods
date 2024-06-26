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
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type PodCreateOptions struct {
	NAME string `help:"Name of server pod" json:"-"`
	ServerCreateCommonConfig
	MEM         string `help:"Memory size MB" metavar:"MEM" json:"-"`
	VcpuCount   int    `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>" json:"vcpu_count" token:"ncpu"`
	AllowDelete *bool  `help:"Unlock server to allow deleting" json:"-"`
	//PortMapping []string `help:"Port mapping of the pod and the format is: host_port=8080,port=80,protocol=<tcp|udp>,host_port_range=<int>-<int>" short-token:"p"`
	Arch             string `help:"image arch" choices:"aarch64|x86_64"`
	AutoStart        bool   `help:"Auto start server after it is created"`
	ShutdownBehavior string `help:"Behavior after VM server shutdown" metavar:"<SHUTDOWN_BEHAVIOR>" choices:"stop|terminate|stop_release_gpu"`
	PodUid           int64  `help:"UID of pod" default:"0"`
	PodGid           int64  `help:"GID of pod" default:"0"`

	ContainerCreateCommonOptions
}

func parsePodPortMappingDetails(input string) (*computeapi.PodPortMapping, error) {
	pm := &computeapi.PodPortMapping{
		Protocol: computeapi.PodPortMappingProtocolTCP,
	}
	for _, seg := range strings.Split(input, ",") {
		info := strings.Split(seg, "=")
		if len(info) != 2 {
			return nil, errors.Errorf("invalid option %s", seg)
		}
		key := info[0]
		val := info[1]
		switch key {
		case "host_port":
			hp, err := strconv.Atoi(val)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid host_port %s", val)
			}
			pm.HostPort = &hp
		case "container_port", "port":
			cp, err := strconv.Atoi(val)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid container_port %s", val)
			}
			pm.ContainerPort = cp
		case "proto", "protocol":
			pm.Protocol = computeapi.PodPortMappingProtocol(val)
		case "host_port_range":
			rangeParts := strings.Split(val, "-")
			if len(rangeParts) != 2 {
				return nil, errors.Errorf("invalid range string %s", val)
			}
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, errors.Wrapf(err, "invalid host_port_range %s", rangeParts[0])
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, errors.Wrapf(err, "invalid host_port_range %s", rangeParts[1])
			}
			pm.HostPortRange = &computeapi.PodPortMappingPortRange{
				Start: start,
				End:   end,
			}
		}
	}
	if pm.ContainerPort == 0 {
		return nil, errors.Error("container_port must specified")
	}
	return pm, nil
}

func ParsePodPortMapping(input string) (*computeapi.PodPortMapping, error) {
	pm, err := parsePodPortMapping(input)
	if err != nil {
		return parsePodPortMappingDetails(input)
	}
	return pm, nil
}

func parsePodPortMapping(input string) (*computeapi.PodPortMapping, error) {
	segs := strings.Split(input, ":")
	parseCtrPart := func(ctrPortPart string) (computeapi.PodPortMappingProtocol, int, error) {
		ctrPortSegs := strings.Split(ctrPortPart, "/")
		if len(ctrPortSegs) > 2 {
			return "", 0, errors.Errorf("wrong format: %s", ctrPortPart)
		}
		ctrPortStr := ctrPortSegs[0]
		ctrPort, err := strconv.Atoi(ctrPortStr)
		if err != nil {
			return "", 0, errors.Wrapf(err, "container_port %s isn't integer", ctrPortStr)
		}
		var protocol computeapi.PodPortMappingProtocol = computeapi.PodPortMappingProtocolTCP
		if len(ctrPortSegs) == 2 {
			switch ctrPortSegs[1] {
			case "tcp":
				protocol = computeapi.PodPortMappingProtocolTCP
			case "udp":
				protocol = computeapi.PodPortMappingProtocolUDP
			//case "sctp":
			//	protocol = computeapi.PodPortMappingProtocolSCTP
			default:
				return "", 0, errors.Wrapf(err, "wrong protocol: %s", ctrPortSegs[1])
			}
		}
		return protocol, ctrPort, nil
	}

	if len(segs) == 1 {
		protocol, ctrPort, err := parseCtrPart(segs[0])
		if err != nil {
			return nil, errors.Wrapf(err, "parse %s", segs[0])
		}
		return &computeapi.PodPortMapping{
			Protocol:      protocol,
			ContainerPort: ctrPort,
		}, nil
	} else if len(segs) == 2 {
		hostPortStr := segs[0]
		hostPort, err := strconv.Atoi(hostPortStr)
		if err != nil {
			return nil, errors.Wrapf(err, "host_port %s isn't integer", hostPortStr)
		}
		ctrPortPart := segs[1]
		protocol, ctrPort, err := parseCtrPart(ctrPortPart)
		if err != nil {
			return nil, errors.Wrapf(err, "parse %s", ctrPortPart)
		}
		return &computeapi.PodPortMapping{
			Protocol:      protocol,
			ContainerPort: ctrPort,
			HostPort:      &hostPort,
		}, nil
	} else {
		return nil, errors.Errorf("wrong format: %s", input)
	}
}

func parseContainerDevice(dev string) (*computeapi.ContainerDevice, error) {
	segs := strings.Split(dev, ":")
	if len(segs) != 3 {
		return nil, errors.Errorf("wrong format: %s", dev)
	}
	return &computeapi.ContainerDevice{
		Type: apis.CONTAINER_DEVICE_TYPE_HOST,
		Host: &computeapi.ContainerHostDevice{
			ContainerPath: segs[1],
			HostPath:      segs[0],
			Permissions:   segs[2],
		},
	}, nil
}

func (o *PodCreateOptions) Params() (*computeapi.ServerCreateInput, error) {
	config, err := o.ServerCreateCommonConfig.Data()
	if err != nil {
		return nil, errors.Wrapf(err, "get ServerCreateCommonConfig.Data")
	}
	config.Hypervisor = computeapi.HYPERVISOR_POD

	/*portMappings := make([]*computeapi.PodPortMapping, 0)
	if len(o.PortMapping) != 0 {
		for _, input := range o.PortMapping {
			pm, err := ParsePodPortMapping(input)
			if err != nil {
				return nil, errors.Wrapf(err, "parse port mapping: %s", input)
			}
			portMappings = append(portMappings, pm)
		}
	}*/

	spec, err := o.getCreateSpec()
	if err != nil {
		return nil, errors.Wrap(err, "get container create spec")
	}

	params := &computeapi.ServerCreateInput{
		ServerConfigs:    config,
		VcpuCount:        o.VcpuCount,
		AutoStart:        o.AutoStart,
		ShutdownBehavior: o.ShutdownBehavior,
		Pod: &computeapi.PodCreateInput{
			//PortMappings: portMappings,
			Containers: []*computeapi.PodContainerCreateInput{
				{
					ContainerSpec: *spec,
				},
			},
			SecurityContext: &computeapi.PodSecurityContext{},
		},
	}

	if o.Uid != 0 {
		params.Pod.SecurityContext.RunAsUser = &o.Uid
	}
	if o.Gid != 0 {
		params.Pod.SecurityContext.RunAsGroup = &o.Gid
	}

	if options.BoolV(o.AllowDelete) {
		disableDelete := false
		params.DisableDelete = &disableDelete
	}
	if regutils.MatchSize(o.MEM) {
		memSize, err := fileutils.GetSizeMb(o.MEM, 'M', 1024)
		if err != nil {
			return nil, err
		}
		params.VmemSize = memSize
	} else {
		return nil, fmt.Errorf("Invalid memory input: %q", o.MEM)
	}
	for idx := range o.IsolatedDevice {
		tmpIdx := idx
		params.Pod.Containers[0].Devices = append(
			params.Pod.Containers[0].Devices,
			&computeapi.ContainerDevice{
				Type:           apis.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
				IsolatedDevice: &computeapi.ContainerIsolatedDevice{Index: &tmpIdx},
			})
	}
	params.OsArch = o.Arch
	params.Name = o.NAME
	return params, nil
}

type PodExecOptions struct {
	ContainerExecOptions
	Scope     string `help:"Scope of containers query" choices:"system|domain|project"`
	Container string `help:"Container name. If omitted, use the first container." short-token:"c"`
}

type PodLogOptions struct {
	ContainerLogOptions
	Scope     string `help:"Scope of containers query" choices:"system|domain|project"`
	Container string `help:"Container name. If omitted, use the first container." short-token:"c"`
}

type PodSetPortMappingOptions struct {
	ServerIdOptions
	PortMapping []string `help:"Port mapping of the pod and the format is: host_port=8080,port=80,protocol=<tcp|udp>,host_port_range=<int>-<int>" short-token:"p"`
}

func (o *PodSetPortMappingOptions) Params() (jsonutils.JSONObject, error) {
	portMappings := make([]*computeapi.PodPortMapping, 0)
	if len(o.PortMapping) != 0 {
		for _, input := range o.PortMapping {
			pm, err := ParsePodPortMapping(input)
			if err != nil {
				return nil, errors.Wrapf(err, "parse port mapping: %s", input)
			}
			portMappings = append(portMappings, pm)
		}
	}
	return jsonutils.Marshal(&computeapi.GuestSetPortMappingsInput{
		PortMappings: portMappings,
	}), nil
}
