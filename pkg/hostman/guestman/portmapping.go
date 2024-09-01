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
		return runtimePm, nil
	}
}
