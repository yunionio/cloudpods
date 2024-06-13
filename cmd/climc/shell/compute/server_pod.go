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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	R(&options.PodCreateOptions{}, "pod-create", "Create a container pod", func(s *mcclient.ClientSession, opts *options.PodCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		if opts.Count > 1 {
			results := modules.Servers.BatchCreate(s, params.JSON(params), opts.Count)
			printBatchResults(results, modules.Servers.GetColumns(s))
		} else {
			server, err := modules.Servers.Create(s, params.JSON(params))
			if err != nil {
				return err
			}
			printObject(server)
		}
		return nil
	})

	getContainerId := func(s *mcclient.ClientSession, scope string, podId string, container string) (string, error) {
		listOpt := map[string]string{
			"guest_id": podId,
		}
		if len(scope) != 0 {
			listOpt["scope"] = scope
		}
		ctrs, err := modules.Containers.List(s, jsonutils.Marshal(listOpt))
		if err != nil {
			return "", errors.Wrapf(err, "list containers by guest_id %s", podId)
		}
		if len(ctrs.Data) == 0 {
			return "", errors.Errorf("count of container is 0")
		}
		var ctrId string
		if container == "" {
			ctrId, _ = ctrs.Data[0].GetString("id")
		} else {
			for _, ctr := range ctrs.Data {
				id, _ := ctr.GetString("id")
				name, _ := ctr.GetString("name")
				if container == id || container == name {
					ctrId, _ = ctr.GetString("id")
					break
				}
			}
		}
		return ctrId, nil
	}

	R(&options.PodExecOptions{}, "pod-exec", "Execute a command in a container", func(s *mcclient.ClientSession, opt *options.PodExecOptions) error {
		ctrId, err := getContainerId(s, opt.Scope, opt.ID, opt.Container)
		if err != nil {
			return err
		}
		return modules.Containers.Exec(s, ctrId, opt.ToAPIInput())
	})

	R(&options.PodLogOptions{}, "pod-log", "Get container log of a pod", func(s *mcclient.ClientSession, opt *options.PodLogOptions) error {
		ctrId, err := getContainerId(s, opt.Scope, opt.ID, opt.Container)
		if err != nil {
			return err
		}
		input, err := opt.ToAPIInput()
		if err != nil {
			return err
		}
		if err := modules.Containers.LogToWriter(s, ctrId, input, os.Stdout); err != nil {
			return errors.Wrap(err, "get container log")
		}
		return nil
	})

	type MigratePortMappingsOptions struct {
		options.ServerIdOptions
		RemovePort []int    `help:"remove port"`
		RemoteIp   []string `help:"remote ips"`
	}
	R(&MigratePortMappingsOptions{}, "pod-migrate-port-mappings", "Migrate port mappings to nic", func(s *mcclient.ClientSession, opt *MigratePortMappingsOptions) error {
		sObj, err := modules.Servers.Get(s, opt.ID, nil)
		if err != nil {
			return err
		}
		id, err := sObj.GetString("id")
		if err != nil {
			return errors.Wrapf(err, "get server id from %s", sObj)
		}
		metadata, err := modules.Servers.GetMetadata(s, id, nil)
		if err != nil {
			return err
		}
		pmStr, err := metadata.GetString(computeapi.POD_METADATA_PORT_MAPPINGS)
		if err != nil {
			return err
		}
		pms, err := jsonutils.ParseString(pmStr)
		if err != nil {
			return err
		}
		pmObjs := []computeapi.PodPortMapping{}
		if err := pms.Unmarshal(&pmObjs); err != nil {
			return errors.Wrapf(err, "unmarshal %s to port_mappings", pms)
		}
		if len(opt.RemovePort) > 0 {
			pp := sets.NewInt(opt.RemovePort...)
			newPmObjs := []computeapi.PodPortMapping{}
			for _, pm := range pmObjs {
				if pp.Has(pm.ContainerPort) {
					continue
				}
				tmpPm := pm
				newPmObjs = append(newPmObjs, tmpPm)
			}
			pmObjs = newPmObjs
		}
		nicPm := make([]*computeapi.GuestPortMapping, len(pmObjs))
		for i, pm := range pmObjs {
			nicPm[i] = &computeapi.GuestPortMapping{
				Protocol:  computeapi.GuestPortMappingProtocol(pm.Protocol),
				Port:      pm.ContainerPort,
				HostPort:  pm.HostPort,
				HostIp:    pm.HostIp,
				RemoteIps: opt.RemoteIp,
			}
			if pm.HostPortRange != nil {
				nicPm[i].HostPortRange = &computeapi.GuestPortMappingPortRange{
					Start: pm.HostPortRange.Start,
					End:   pm.HostPortRange.End,
				}
			}
		}
		params := jsonutils.Marshal(map[string]interface{}{
			"scope":   "system",
			"details": true,
		})
		sNets, err := modules.Servernetworks.ListDescendent(s, id, params)
		if err != nil {
			return errors.Wrap(err, "list server networks")
		}
		if len(sNets.Data) == 0 {
			return errors.Errorf("no server networks found")
		}
		firstNet := sNets.Data[0]
		sNet := new(computeapi.GuestnetworkDetails)
		if err := firstNet.Unmarshal(sNet); err != nil {
			return errors.Wrap(err, "unmarshal to guestnetwork details")
		}

		updateData := jsonutils.Marshal(map[string]interface{}{
			"port_mappings": nicPm,
		})
		obj, err := modules.Servernetworks.Update(s, sNet.GuestId, sNet.NetworkId, nil, updateData)
		if err != nil {
			return errors.Wrap(err, "update server networks")
		}
		printObject(obj)

		return nil
	})
}
