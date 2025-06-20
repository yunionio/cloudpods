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
	"context"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/netutils2/getport"
)

var (
	allocatePortLock sync.Mutex
)

type IPortMappingManager interface {
	AllocateGuestPortMappings(ctx context.Context, userCred mcclient.TokenCredential, guest GuestRuntimeInstance) error
}

type portMappingManager struct {
	manager *SGuestManager
}

func NewPortMappingManager(manager *SGuestManager) IPortMappingManager {
	return &portMappingManager{
		manager: manager,
	}
}

func (m *portMappingManager) GetGuestPortMappings(guest GuestRuntimeInstance) map[string]compute.GuestPortMappings {
	nics := guest.GetSourceDesc().Nics
	pms := make(map[string]compute.GuestPortMappings)
	for _, nic := range nics {
		if len(nic.PortMappings) == 0 {
			continue
		}
		pms[nic.NetId] = nic.PortMappings
	}
	return pms
}

func (m *portMappingManager) IsGuestHasPortMapping(guest GuestRuntimeInstance) bool {
	return len(m.GetGuestPortMappings(guest)) == 0
}

func (m *portMappingManager) AllocateGuestPortMappings(ctx context.Context, userCred mcclient.TokenCredential, guest GuestRuntimeInstance) error {
	allocatePortLock.Lock()
	defer allocatePortLock.Unlock()

	for idx, nic := range guest.GetDesc().Nics {
		if len(nic.PortMappings) == 0 {
			continue
		}
		newPms, err := m.allocatePortMappings(guest, nic.PortMappings)
		if err != nil {
			return errors.Wrapf(err, "allocateGuestPortMapping for nic %d: %s", idx, jsonutils.Marshal(nic.PortMappings))
		}
		// update allocated port mappings
		if err := m.setPortMappings(ctx, userCred, guest, idx, newPms); err != nil {
			return errors.Wrapf(err, "setPortMappings for nic %d", idx)
		}
	}
	return nil
}

func (m *portMappingManager) setPortMappings(ctx context.Context, userCred mcclient.TokenCredential, gst GuestRuntimeInstance, nicIdx int, pms compute.GuestPortMappings) error {
	// update desc
	desc := gst.GetDesc()
	nic := desc.Nics[nicIdx]
	nic.PortMappings = pms
	desc.Nics[nicIdx] = nic

	// update port mapping info to controller
	body := jsonutils.Marshal(map[string]interface{}{
		"port_mappings": pms,
	})
	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Servernetworks.Update(session, gst.GetId(), nic.NetId, nil, body); err != nil {
		return errors.Wrapf(err, "update server %s network %s with port_mappings %s", gst.GetId(), nic.NetId, body.String())
	}

	// save desc
	gst.SetDesc(desc)
	return SaveDesc(gst, desc)
}

func (m *portMappingManager) getOtherGuests(gst GuestRuntimeInstance) []GuestRuntimeInstance {
	others := make([]GuestRuntimeInstance, 0)
	m.manager.Servers.Range(func(id, value interface{}) bool {
		if id == gst.GetId() {
			return true
		}
		ins := value.(GuestRuntimeInstance)
		others = append(others, ins)
		return true
	})
	return others
}

func (m *portMappingManager) getGuestFlattenPortMappings(guest GuestRuntimeInstance) compute.GuestPortMappings {
	ret := make([]*compute.GuestPortMapping, 0)
	pms := m.GetGuestPortMappings(guest)
	for _, pm := range pms {
		for _, p := range pm {
			ret = append(ret, p)
		}
	}
	return ret
}

func (m *portMappingManager) getOtherGuestsUsedPorts(gst GuestRuntimeInstance) (map[compute.GuestPortMappingProtocol]sets.Int, error) {
	others := m.getOtherGuests(gst)
	ret := make(map[compute.GuestPortMappingProtocol]sets.Int)
	for _, ins := range others {
		pms := m.getGuestFlattenPortMappings(ins)
		for _, pm := range pms {
			ps, ok := ret[pm.Protocol]
			if !ok {
				ps = sets.NewInt()
			}
			if pm.HostPort == nil {
				//return nil, errors.Errorf("guest (%s/%s) portmap %s has nil host port", ins.GetId(), ins.GetName(), jsonutils.Marshal(pm))
				log.Warningf("%s", errors.Errorf("guest (%s/%s) portmap %s has nil host port", ins.GetId(), ins.GetName(), jsonutils.Marshal(pm)))
				continue
			}
			ps.Insert(*pm.HostPort)
			ret[pm.Protocol] = ps
		}
	}
	return ret, nil
}

func (m *portMappingManager) allocatePortMappings(gst GuestRuntimeInstance, input compute.GuestPortMappings) (compute.GuestPortMappings, error) {
	result := make([]*compute.GuestPortMapping, len(input))
	allocPorts := make(map[compute.GuestPortMappingProtocol]sets.Int)

	// 检查是否有需要按规则分配的端口映射
	hasRuleMapping := false
	for _, pm := range input {
		if pm.Rule != nil && pm.Rule.FirstPortOffset != nil {
			hasRuleMapping = true
			break
		}
	}

	if hasRuleMapping {
		// 如果有规则映射，需要先找到第一个空闲端口，然后按偏移量分配
		return m.allocatePortMappingsWithRule(gst, input, allocPorts)
	}

	// 原有的分配逻辑
	for idx := range input {
		data := input[idx]
		if _, ok := allocPorts[data.Protocol]; !ok {
			allocPorts[data.Protocol] = sets.NewInt()
		}
		pm, err := m.allocatePortMapping(gst, data, allocPorts)
		if err != nil {
			return nil, errors.Wrapf(err, "get port mapping %s", jsonutils.Marshal(input[idx]))
		}
		result[idx] = pm
		allocPorts[data.Protocol].Insert(*pm.HostPort)
	}
	return result, nil
}

func (m *portMappingManager) allocatePortMappingsWithRule(gst GuestRuntimeInstance, input compute.GuestPortMappings, allocPorts map[compute.GuestPortMappingProtocol]sets.Int) (compute.GuestPortMappings, error) {
	result := make([]*compute.GuestPortMapping, len(input))

	// 按协议分组，分别处理
	indices := make([]*compute.GuestPortMapping, 0)
	for idx, pm := range input {
		if pm.Rule != nil && pm.Rule.FirstPortOffset != nil {
			indices = append(indices, input[idx])
		}
	}

	// 为每个协议组分配端口
	if err := m.allocateProtocolGroupWithRule(gst, input, result, indices, allocPorts); err != nil {
		return nil, errors.Wrapf(err, "allocate portmappings with rule: %s", jsonutils.Marshal(indices))
	}

	// 处理没有规则的端口映射
	for idx, pm := range input {
		if pm.Rule == nil || pm.Rule.FirstPortOffset == nil {
			if _, ok := allocPorts[pm.Protocol]; !ok {
				allocPorts[pm.Protocol] = sets.NewInt()
			}
			allocatedPm, err := m.allocatePortMapping(gst, pm, allocPorts)
			if err != nil {
				return nil, errors.Wrapf(err, "get port mapping %s", jsonutils.Marshal(pm))
			}
			result[idx] = allocatedPm
			allocPorts[pm.Protocol].Insert(*allocatedPm.HostPort)
		}
	}

	return result, nil
}

func (m *portMappingManager) allocateProtocolGroupWithRule(gst GuestRuntimeInstance, input compute.GuestPortMappings, result compute.GuestPortMappings, indices []*compute.GuestPortMapping, allocPorts map[compute.GuestPortMappingProtocol]sets.Int) error {
	// 获取其他虚拟机已使用的端口
	otherPorts, err := m.getOtherGuestsUsedPorts(gst)
	if err != nil {
		return errors.Wrap(err, "getOtherGuestsUsedPorts")
	}

	// 获取当前协议已分配的端口
	usedPorts := map[compute.GuestPortMappingProtocol]sets.Int{
		compute.GuestPortMappingProtocolTCP: sets.NewInt(),
		compute.GuestPortMappingProtocolUDP: sets.NewInt(),
	}
	for proto, ports := range otherPorts {
		usedPorts[proto].Insert(ports.List()...)
	}
	for proto, allocPortsSet := range allocPorts {
		if ports, ok := usedPorts[proto]; ok {
			ports.Insert(allocPortsSet.List()...)
			usedPorts[proto] = ports
		} else {
			usedPorts[proto] = sets.NewInt(allocPortsSet.List()...)
		}
	}

	// 确定端口范围
	start := compute.GUEST_PORT_MAPPING_RANGE_START
	end := compute.GUEST_PORT_MAPPING_RANGE_END

	// 尝试不同的 basePort，直到找到满足所有规则要求的端口
	success := false
	for basePort := start; basePort <= end; basePort++ {
		// 检查这个 basePort 是否能满足所有规则要求
		if m.canAllocateWithBasePort(basePort, input, indices, usedPorts) {
			// 分配端口
			if err := m.allocateWithBasePort(basePort, input, result, indices, usedPorts, allocPorts); err != nil {
				// 如果分配失败，继续尝试下一个 basePort
				continue
			}
			success = true
			break
		}
	}

	if !success {
		return errors.Errorf("cannot find suitable base port for protocol %s in range %d-%d", indices[0].Protocol, start, end)
	}

	return nil
}

func (m *portMappingManager) checkPortIsUsed(port int, protocol compute.GuestPortMappingProtocol, usedPorts map[compute.GuestPortMappingProtocol]sets.Int) bool {
	portProtocol := getport.TCP
	if protocol == compute.GuestPortMappingProtocolUDP {
		portProtocol = getport.UDP
	}
	if _, ok := usedPorts[protocol]; !ok {
		usedPorts[protocol] = sets.NewInt()
	}
	return usedPorts[protocol].Has(port) || getport.IsPortUsed(portProtocol, "", port)
}

func (m *portMappingManager) canAllocateWithBasePort(basePort int, input compute.GuestPortMappings, indices []*compute.GuestPortMapping, usedPorts map[compute.GuestPortMappingProtocol]sets.Int) bool {
	baseProtocol := indices[0].Protocol

	// 检查 basePort 本身是否可用
	if m.checkPortIsUsed(basePort, baseProtocol, usedPorts) {
		return false
	}

	// 检查所有设置了规则的端口是否都可用
	for _, pm := range indices {
		offset := *pm.Rule.FirstPortOffset
		targetPort := basePort + offset

		// 检查目标端口是否在范围内
		if targetPort > compute.GUEST_PORT_MAPPING_RANGE_END {
			return false
		}

		// 检查目标端口是否已被使用
		if m.checkPortIsUsed(targetPort, pm.Protocol, usedPorts) {
			return false
		}
	}

	return true
}

func (m *portMappingManager) allocateWithBasePort(basePort int, input compute.GuestPortMappings, result compute.GuestPortMappings, indices []*compute.GuestPortMapping, usedPorts, allocPorts map[compute.GuestPortMappingProtocol]sets.Int) error {
	// 分配所有设置了规则的端口
	for idx, _ := range indices {
		pm := input[idx]
		offset := *pm.Rule.FirstPortOffset
		targetPort := basePort + offset

		// 再次检查端口可用性（双重检查）
		if m.checkPortIsUsed(targetPort, pm.Protocol, usedPorts) {
			return errors.Errorf("port %d is not available for protocol %s", targetPort, pm.Protocol)
		}

		// 创建分配的端口映射
		runtimePm := &compute.GuestPortMapping{}
		if err := jsonutils.Marshal(pm).Unmarshal(runtimePm); err != nil {
			return errors.Wrap(err, "unmarshal to runtime port mapping")
		}

		runtimePm.HostPort = &targetPort
		if runtimePm.Port == -1 {
			runtimePm.Port = targetPort
		}
		result[idx] = runtimePm

		// 更新已使用端口集合
		usedPorts[pm.Protocol].Insert(targetPort)
		if _, ok := allocPorts[pm.Protocol]; !ok {
			allocPorts[pm.Protocol] = sets.NewInt()
		}
		allocPorts[pm.Protocol].Insert(targetPort)
	}

	return nil
}

func (m *portMappingManager) allocatePortMapping(gst GuestRuntimeInstance, pm *compute.GuestPortMapping, allocPorts map[compute.GuestPortMappingProtocol]sets.Int) (*compute.GuestPortMapping, error) {
	otherPorts, err := m.getOtherGuestsUsedPorts(gst)
	if err != nil {
		return nil, errors.Wrap(err, "getOtherPodsUsedPorts")
	}

	// copy to runtime port mapping
	runtimePm := &compute.GuestPortMapping{}
	if err := jsonutils.Marshal(pm).Unmarshal(runtimePm); err != nil {
		return nil, errors.Wrap(err, "unmarshal to runtime port mapping")
	}

	portProtocol := getport.TCP
	switch pm.Protocol {
	case compute.GuestPortMappingProtocolTCP:
		portProtocol = getport.TCP
	case compute.GuestPortMappingProtocolUDP:
		portProtocol = getport.UDP
	default:
		return nil, errors.Errorf("invalid protocol: %q", pm.Protocol)
	}

	if pm.HostPort != nil {
		runtimePm.HostPort = pm.HostPort
		if getport.IsPortUsed(portProtocol, "", *pm.HostPort) {
			return nil, httperrors.NewInputParameterError("host_port %d is used", *pm.HostPort)
		}
		usedPorts, ok := otherPorts[pm.Protocol]
		if ok {
			if usedPorts.Has(*pm.HostPort) {
				return nil, errors.Errorf("%s host_port %d is already used", pm.Protocol, *pm.HostPort)
			}
		}
		allocProtoPorts, ok := allocPorts[pm.Protocol]
		if ok {
			if allocProtoPorts.Has(*pm.HostPort) {
				return nil, errors.Errorf("%s host_port %d is already allocated", pm.Protocol, *pm.HostPort)
			}
		}
		if runtimePm.Port == -1 {
			runtimePm.Port = *pm.HostPort
		}
		return runtimePm, nil
	} else {
		start := compute.GUEST_PORT_MAPPING_RANGE_START
		end := compute.GUEST_PORT_MAPPING_RANGE_END
		if pm.HostPortRange != nil {
			start = pm.HostPortRange.Start
			end = pm.HostPortRange.End
		}
		otherPodPorts, ok := otherPorts[pm.Protocol]
		if !ok {
			otherPodPorts = sets.NewInt()
		}
		allocProtoPorts, ok := allocPorts[pm.Protocol]
		if ok {
			otherPodPorts.Insert(allocProtoPorts.List()...)
		}
		portResult, err := getport.GetPortByRangeBySets(portProtocol, start, end, otherPodPorts)
		if err != nil {
			return nil, errors.Wrapf(err, "listen %s port inside %d and %d", pm.Protocol, start, end)
		}
		runtimePm.HostPort = &portResult.Port
		if runtimePm.Port == -1 {
			runtimePm.Port = portResult.Port
		}
		return runtimePm, nil
	}
}
