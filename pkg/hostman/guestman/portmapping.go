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
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2/getport"
)

var (
	allocatePortLock sync.Mutex
)

type IPortMappingManager interface {
	AllocateGuestPortMappings(ctx context.Context, userCred mcclient.TokenCredential, guest GuestRuntimeInstance, guestDesc *desc.SGuestDesc) error
	SetGuestNicPortMappings(ctx context.Context, userCred mcclient.TokenCredential, guest GuestRuntimeInstance, input *compute.ServerSetPortMappingInput) error
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

func (m *portMappingManager) AllocateGuestPortMappings(ctx context.Context, userCred mcclient.TokenCredential, guest GuestRuntimeInstance, guestDesc *desc.SGuestDesc) error {
	allocatePortLock.Lock()
	defer allocatePortLock.Unlock()

	if guestDesc == nil {
		return nil
	}
	for idx, nic := range guestDesc.Nics {
		if len(nic.PortMappings) == 0 {
			continue
		}
		newPms, err := m.allocatePortMappings(guest, nic.PortMappings)
		if err != nil {
			return errors.Wrapf(err, "allocateGuestPortMapping for nic %d: %s", idx, jsonutils.Marshal(nic.PortMappings))
		}
		nic.PortMappings = newPms
		guestDesc.Nics[idx] = nic
		// update allocated port mappings
		if err := m.setPortMappings(ctx, userCred, guest, guestDesc, idx, newPms); err != nil {
			return errors.Wrapf(err, "setPortMappings for nic %d", idx)
		}
	}
	return nil
}

// SetGuestNicPortMappings 设置虚机指定网卡的端口映射：
//   - 已有 host_port 的映射保持原值，但仍做占用校验（本虚机自身已占用的端口除外，否则重复提交同一映射会误报被占用）
//   - 缺少 host_port 的映射优先复用同一容器端口上一次分配的 host_port，否则在宿主机端口范围内重新分配
//   - 最终结果回写 region（带 no_sync，避免递归触发 region -> host 的同步）
func (m *portMappingManager) SetGuestNicPortMappings(ctx context.Context, userCred mcclient.TokenCredential, guest GuestRuntimeInstance, input *compute.ServerSetPortMappingInput) error {
	allocatePortLock.Lock()
	defer allocatePortLock.Unlock()

	if input == nil {
		return nil
	}
	guestDesc := guest.GetSourceDesc()
	if guestDesc == nil {
		return errors.Errorf("guest %s source desc is nil", guest.GetId())
	}
	nicIdx := -1
	for i := range guestDesc.Nics {
		if matchGuestNetworkInfo(guestDesc.Nics[i], &input.ServerNetworkInfo) {
			nicIdx = i
			break
		}
	}
	if nicIdx < 0 {
		return errors.Errorf("cannot find nic %s of guest %s", jsonutils.Marshal(&input.ServerNetworkInfo), guest.GetId())
	}
	oldPms := guestDesc.Nics[nicIdx].PortMappings
	selfPorts := m.getGuestSelfUsedHostPorts(guest)
	allocPorts := make(map[compute.GuestPortMappingProtocol]sets.Int)

	result := make(compute.GuestPortMappings, 0, len(input.PortMappings))
	for i := range input.PortMappings {
		pm := input.PortMappings[i]
		newPm := *pm
		if newPm.HostPort == nil {
			// 复用同一容器端口上一次分配的 host_port，避免每次编辑都重新分配导致端口漂移
			if old := findPortMappingByPort(oldPms, pm); old != nil && old.HostPort != nil {
				hostPort := *old.HostPort
				newPm.HostPort = &hostPort
			}
		}
		// allocatePortMapping 要求调用方先初始化该协议的已分配端口集合
		if _, ok := allocPorts[newPm.Protocol]; !ok {
			allocPorts[newPm.Protocol] = sets.NewInt()
		}
		// 统一走 allocatePortMapping：已有 host_port 的会做占用校验（本虚机自身占用的端口除外），
		// 缺失 host_port 的会在宿主机端口范围内分配
		allocated, err := m.allocatePortMapping(guest, &newPm, allocPorts, selfPorts)
		if err != nil {
			return errors.Wrapf(err, "allocatePortMapping %s", jsonutils.Marshal(pm))
		}
		allocPorts[allocated.Protocol].Insert(*allocated.HostPort)
		result = append(result, allocated)
	}
	if err := m.setNicPortMappings(ctx, userCred, guest, nicIdx, result); err != nil {
		return errors.Wrap(err, "setNicPortMappings")
	}
	return nil
}

// matchGuestNetworkInfo 判断网卡是否与指定的 ip / ip6 / mac / index 匹配
func matchGuestNetworkInfo(nic *desc.SGuestNetwork, info *compute.ServerNetworkInfo) bool {
	switch {
	case len(info.IpAddr) > 0:
		return nic.Ip == info.IpAddr
	case len(info.Ip6Addr) > 0:
		return nic.Ip6 == info.Ip6Addr
	case len(info.Mac) > 0:
		return netutils2.MacEqual(nic.Mac, info.Mac)
	default:
		return nic.Index == info.Index
	}
}

func findPortMappingByPort(pms compute.GuestPortMappings, pm *compute.GuestPortMapping) *compute.GuestPortMapping {
	for _, p := range pms {
		if p.Port == pm.Port && p.Protocol == pm.Protocol {
			return p
		}
	}
	return nil
}

// getGuestSelfUsedHostPorts 本虚机（所有网卡）当前已占用的宿主机端口
func (m *portMappingManager) getGuestSelfUsedHostPorts(guest GuestRuntimeInstance) sets.Int {
	ret := sets.NewInt()
	for _, pm := range m.getGuestFlattenPortMappings(guest) {
		if pm.HostPort != nil {
			ret.Insert(*pm.HostPort)
		}
	}
	return ret
}

// setNicPortMappings 把网卡的 port_mappings 写回 region，并更新宿主机本地保存的 source desc
func (m *portMappingManager) setNicPortMappings(ctx context.Context, userCred mcclient.TokenCredential, gst GuestRuntimeInstance, nicIdx int, pms compute.GuestPortMappings) error {
	guestDesc := gst.GetSourceDesc()
	nic := guestDesc.Nics[nicIdx]
	nic.PortMappings = pms
	guestDesc.Nics[nicIdx] = nic

	// update port mapping info to controller
	// no_sync=true：这里只是把宿主机设置/分配好的 port_mappings 回写 region，
	// 不应再触发 region -> host 的同步，避免递归
	body := jsonutils.Marshal(map[string]interface{}{
		"port_mappings": pms,
		"no_sync":       true,
	})
	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Servernetworks.Update(session, gst.GetId(), nic.NetId, nil, body); err != nil {
		return errors.Wrapf(err, "update server %s network %s with port_mappings %s", gst.GetId(), nic.NetId, body.String())
	}

	// save desc
	return SaveDesc(gst, guestDesc)
}

func (m *portMappingManager) setPortMappings(ctx context.Context, userCred mcclient.TokenCredential, gst GuestRuntimeInstance, guestDesc *desc.SGuestDesc, nicIdx int, pms compute.GuestPortMappings) error {
	// update desc
	nic := guestDesc.Nics[nicIdx]
	nic.PortMappings = pms
	guestDesc.Nics[nicIdx] = nic

	// update port mapping info to controller
	// no_sync=true：这里只是把宿主机分配到的 host_port 等回写 region，
	// 不应再触发 region -> host 的同步，避免递归
	body := jsonutils.Marshal(map[string]interface{}{
		"port_mappings": pms,
		"no_sync":       true,
	})
	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Servernetworks.Update(session, gst.GetId(), nic.NetId, nil, body); err != nil {
		return errors.Wrapf(err, "update server %s network %s with port_mappings %s", gst.GetId(), nic.NetId, body.String())
	}

	// save desc
	gst.SetDesc(guestDesc)
	return SaveDesc(gst, guestDesc)
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
	// 本虚机自身已占用的宿主机端口，探测占用时需要排除，否则会把自己的端口误判为被占用
	selfPorts := m.getGuestSelfUsedHostPorts(gst)

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
		return m.allocatePortMappingsWithRule(gst, input, allocPorts, selfPorts)
	}

	// 原有的分配逻辑
	for idx := range input {
		data := input[idx]
		if _, ok := allocPorts[data.Protocol]; !ok {
			allocPorts[data.Protocol] = sets.NewInt()
		}
		pm, err := m.allocatePortMapping(gst, data, allocPorts, selfPorts)
		if err != nil {
			return nil, errors.Wrapf(err, "get port mapping %s", jsonutils.Marshal(input[idx]))
		}
		result[idx] = pm
		allocPorts[data.Protocol].Insert(*pm.HostPort)
	}
	return result, nil
}

func (m *portMappingManager) allocatePortMappingsWithRule(gst GuestRuntimeInstance, input compute.GuestPortMappings, allocPorts map[compute.GuestPortMappingProtocol]sets.Int, selfPorts sets.Int) (compute.GuestPortMappings, error) {
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
			allocatedPm, err := m.allocatePortMapping(gst, pm, allocPorts, selfPorts)
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

	// 确定端口范围（从 hostman options 读取）
	start := options.HostOptions.PortMappingRangeStart
	end := options.HostOptions.PortMappingRangeEnd

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

		// 检查目标端口是否在配置的范围内
		if targetPort > options.HostOptions.PortMappingRangeEnd {
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

func (m *portMappingManager) allocatePortMapping(gst GuestRuntimeInstance, pm *compute.GuestPortMapping, allocPorts map[compute.GuestPortMappingProtocol]sets.Int, selfPorts sets.Int) (*compute.GuestPortMapping, error) {
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
		// 该端口可能是本虚机上一次已分配并正在转发的端口，不能因此判定为"被占用"
		if !selfPorts.Has(*pm.HostPort) && getport.IsPortUsed(portProtocol, "", *pm.HostPort) {
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
		start := options.HostOptions.PortMappingRangeStart
		end := options.HostOptions.PortMappingRangeEnd
		if pm.HostPortRange != nil {
			if pm.HostPortRange.Start > start {
				start = pm.HostPortRange.Start
			}
			if pm.HostPortRange.End < end {
				end = pm.HostPortRange.End
			}
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
