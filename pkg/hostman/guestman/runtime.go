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
	"fmt"
	"io/ioutil"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type GuestRuntimeInstance interface {
	GetHypervisor() string
	GetName() string
	GetInitialId() string
	GetId() string
	IsValid() bool
	HomeDir() string
	GetDesc() *desc.SGuestDesc
	SetDesc(guestDesc *desc.SGuestDesc)
	GetSourceDesc() *desc.SGuestDesc
	SetSourceDesc(guestDesc *desc.SGuestDesc)
	GetDescFilePath() string
	NicTrafficRecordPath() string
	DeployFs(ctx context.Context, userCred mcclient.TokenCredential, deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error)
	GetSourceDescFilePath() string
	IsRunning() bool
	IsStopped() bool
	IsSuspend() bool
	IsLoaded() bool
	GetNicDescMatch(mac, ip, port, bridge string) *desc.SGuestNetwork
	CleanGuest(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	CleanDirtyGuest(ctx context.Context) error
	ImportServer(pendingDelete bool)
	ExitCleanup(bool)

	HandleGuestStatus(ctx context.Context, status string, body *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	HandleGuestStart(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject) (jsonutils.JSONObject, error)

	HandleStop(ctx context.Context, timeout int64) error

	LoadDesc() error
	PostLoad(m *SGuestManager) error
	SyncConfig(ctx context.Context, guestDesc *desc.SGuestDesc, fwOnly bool) (jsonutils.JSONObject, error)
	DoSnapshot(ctx context.Context, params *SDiskSnapshot) (jsonutils.JSONObject, error)
	DeleteSnapshot(ctx context.Context, params *SDeleteDiskSnapshot) (jsonutils.JSONObject, error)
}

type sBaseGuestInstance struct {
	Id      string
	manager *SGuestManager
	// runtime description, generate from source desc
	Desc *desc.SGuestDesc
	// source description, input from region
	SourceDesc *desc.SGuestDesc
	Hypervisor string
}

func newBaseGuestInstance(id string, manager *SGuestManager, hypervisor string) *sBaseGuestInstance {
	return &sBaseGuestInstance{
		Id:         id,
		manager:    manager,
		Hypervisor: hypervisor,
	}
}

func (s *sBaseGuestInstance) GetHypervisor() string {
	return s.Hypervisor
}

func (s *sBaseGuestInstance) IsValid() bool {
	return s.Desc != nil && s.Desc.Uuid != ""
}

func (s *sBaseGuestInstance) GetInitialId() string {
	return s.Id
}

func (s *sBaseGuestInstance) GetId() string {
	return s.Desc.Uuid
}

func (s *sBaseGuestInstance) GetName() string {
	return fmt.Sprintf("%s(%s)", s.Desc.Name, s.Desc.Uuid)
}

func (b *sBaseGuestInstance) HomeDir() string {
	return path.Join(b.manager.ServersPath, b.Id)
}

func (s *sBaseGuestInstance) GetDescFilePath() string {
	return path.Join(s.HomeDir(), "desc")
}

func (s *sBaseGuestInstance) GetDesc() *desc.SGuestDesc {
	return s.Desc
}

func (s *sBaseGuestInstance) SetDesc(guestDesc *desc.SGuestDesc) {
	s.Desc = guestDesc
}

func (s *sBaseGuestInstance) GetSourceDesc() *desc.SGuestDesc {
	return s.SourceDesc
}

func (s *sBaseGuestInstance) SetSourceDesc(guestDesc *desc.SGuestDesc) {
	s.SourceDesc = guestDesc
}

func (s *sBaseGuestInstance) GetSourceDescFilePath() string {
	return path.Join(s.HomeDir(), "source-desc")
}

func (s *sBaseGuestInstance) NicTrafficRecordPath() string {
	return path.Join(s.HomeDir(), "nic_traffic.json")
}

func (s *sBaseGuestInstance) IsLoaded() bool {
	return s.Desc != nil
}

func (s *sBaseGuestInstance) IsDaemon() bool {
	return s.Desc.IsDaemon
}

func (s *sBaseGuestInstance) GetNicDescMatch(mac, ip, port, bridge string) *desc.SGuestNetwork {
	nics := s.Desc.Nics
	for _, nic := range nics {
		if bridge == "" && nic.Bridge != "" && nic.Bridge == options.HostOptions.OvnIntegrationBridge {
			continue
		}
		if (len(mac) == 0 || netutils2.MacEqual(nic.Mac, mac)) &&
			(len(ip) == 0 || nic.Ip == ip) &&
			(len(port) == 0 || nic.Ifname == port) &&
			(len(bridge) == 0 || nic.Bridge == bridge) {
			return nic
		}
	}
	return nil
}

func LoadGuestCpuset(m *SGuestManager, s GuestRuntimeInstance) error {
	guestDesc := s.GetDesc()
	if s.IsRunning() {
		m.cpuSet.Lock.Lock()
		defer m.cpuSet.Lock.Unlock()
		for _, vcpuPin := range guestDesc.VcpuPin {
			pcpuSet, err := cpuset.Parse(vcpuPin.Pcpus)
			if err != nil {
				log.Errorf("failed parse %s pcpus: %s", s.GetName(), vcpuPin.Pcpus)
				continue
			}
			vcpuSet, err := cpuset.Parse(vcpuPin.Vcpus)
			if err != nil {
				log.Errorf("failed parse %s vcpus: %s", s.GetName(), vcpuPin.Vcpus)
				continue
			}
			m.cpuSet.LoadCpus(pcpuSet.ToSlice(), vcpuSet.Size())
		}
		for _, numaCpuset := range guestDesc.CpuNumaPin {
			pcpus := make([]int, 0)
			for i := range numaCpuset.VcpuPin {
				pcpus = append(pcpus, numaCpuset.VcpuPin[i].Pcpu)
			}

			m.cpuSet.LoadNumaCpus(numaCpuset.SizeMB, int(*numaCpuset.NodeId), pcpus, len(numaCpuset.VcpuPin))
		}
	}
	return nil
}

func ReleaseCpuNumaPin(m *SGuestManager, cpuNumaPin []*desc.SCpuNumaPin) {
	if !m.numaAllocate {
		return
	}

	for _, numaCpus := range cpuNumaPin {
		pcpus := make([]int, 0)
		for i := range numaCpus.VcpuPin {
			pcpus = append(pcpus, numaCpus.VcpuPin[i].Pcpu)
		}
		m.cpuSet.ReleaseNumaCpus(numaCpus.SizeMB, int(*numaCpus.NodeId), pcpus, len(numaCpus.VcpuPin))
	}
}

func ReleaseGuestCpuset(m *SGuestManager, s GuestRuntimeInstance) {
	m.cpuSet.Lock.Lock()
	defer m.cpuSet.Lock.Unlock()
	guestDesc := s.GetDesc()
	ReleaseCpuNumaPin(m, guestDesc.CpuNumaPin)

	for _, vcpuPin := range guestDesc.VcpuPin {
		pcpuSet, err := cpuset.Parse(vcpuPin.Pcpus)
		if err != nil {
			log.Errorf("failed parse %s pcpus: %s", s.GetName(), vcpuPin.Pcpus)
			continue
		}
		vcpuSet, err := cpuset.Parse(vcpuPin.Vcpus)
		if err != nil {
			log.Errorf("failed parse %s vcpus: %s", s.GetName(), vcpuPin.Vcpus)
			continue
		}
		m.cpuSet.ReleaseCpus(pcpuSet.ToSlice(), vcpuSet.Size())
	}
	guestDesc.VcpuPin = nil
	guestDesc.CpuNumaPin = nil
	SaveLiveDesc(s, guestDesc)
}

type GuestRuntimeManager struct {
}

func NewGuestRuntimeManager() *GuestRuntimeManager {
	return new(GuestRuntimeManager)
}

func (f *GuestRuntimeManager) NewRuntimeInstance(id string, manager *SGuestManager, hypervisor string) GuestRuntimeInstance {
	switch hypervisor {
	case computeapi.HYPERVISOR_KVM, "":
		return NewKVMGuestInstance(id, manager)
	case computeapi.HYPERVISOR_POD:
		return newPodGuestInstance(id, manager)
	}
	log.Fatalf("Invalid hypervisor for runtime: %q", hypervisor)
	return nil
}

func PrepareDir(s GuestRuntimeInstance) error {
	output, err := procutils.NewCommand("mkdir", "-p", s.HomeDir()).Output()
	if err != nil {
		return errors.Wrapf(err, "mkdir %s failed: %s", s.HomeDir(), output)
	}
	return nil
}

func (f *GuestRuntimeManager) CreateFromDesc(s GuestRuntimeInstance, desc *desc.SGuestDesc) error {
	if err := PrepareDir(s); err != nil {
		return errors.Errorf("Failed to create server dir %s", desc.Uuid)
	}
	return SaveDesc(s, desc)
}

func (s *sBaseGuestInstance) GetVpcNIC() *desc.SGuestNetwork {
	for _, nic := range s.Desc.Nics {
		if nic.Vpc.Provider == computeapi.VPC_PROVIDER_OVN {
			if nic.Ip != "" {
				return nic
			}
		}
	}
	return nil
}

func LoadDesc(s GuestRuntimeInstance) error {
	descPath := s.GetDescFilePath()
	descStr, err := ioutil.ReadFile(descPath)
	if err != nil {
		return errors.Wrap(err, "read desc")
	}

	var (
		srcDescStr  []byte
		srcDescPath = s.GetSourceDescFilePath()
	)
	if !fileutils2.Exists(srcDescPath) {
		err = fileutils2.FilePutContents(srcDescPath, string(descStr), false)
		if err != nil {
			return errors.Wrap(err, "save source desc")
		}
		srcDescStr = descStr
	} else {
		srcDescStr, err = ioutil.ReadFile(srcDescPath)
		if err != nil {
			return errors.Wrap(err, "read source desc")
		}
	}

	// parse source desc
	srcGuestDesc := new(desc.SGuestDesc)
	jsonSrcDesc, err := jsonutils.Parse(srcDescStr)
	if err != nil {
		return errors.Wrap(err, "json parse source desc")
	}
	err = jsonSrcDesc.Unmarshal(srcGuestDesc)
	if err != nil {
		return errors.Wrap(err, "unmarshal source desc")
	}
	s.SetSourceDesc(srcGuestDesc)

	// parse desc
	guestDesc := new(desc.SGuestDesc)
	jsonDesc, err := jsonutils.Parse(descStr)
	if err != nil {
		return errors.Wrap(err, "json parse desc")
	}
	err = jsonDesc.Unmarshal(guestDesc)
	if err != nil {
		return errors.Wrap(err, "unmarshal desc")
	}
	s.SetDesc(guestDesc)
	return nil
}

func SaveDesc(s GuestRuntimeInstance, guestDesc *desc.SGuestDesc) error {
	s.SetSourceDesc(guestDesc)
	// fill in ovn vpc nic bridge field
	for _, nic := range s.GetSourceDesc().Nics {
		if nic.Bridge == "" {
			nic.Bridge = getNicBridge(nic)
		}
	}

	if err := fileutils2.FilePutContents(
		s.GetSourceDescFilePath(), jsonutils.Marshal(s.GetSourceDesc()).String(), false,
	); err != nil {
		log.Errorf("save source desc failed %s", err)
		return errors.Wrap(err, "source save desc")
	}

	if !s.IsRunning() { // if guest not running, sync live desc
		liveDesc := new(desc.SGuestDesc)
		if err := jsonutils.Marshal(s.GetSourceDesc()).Unmarshal(liveDesc); err != nil {
			return errors.Wrap(err, "unmarshal live desc")
		}
		return SaveLiveDesc(s, liveDesc)
	}
	return nil
}

func SaveLiveDesc(s GuestRuntimeInstance, guestDesc *desc.SGuestDesc) error {
	s.SetDesc(guestDesc)

	defaultGwCnt := 0
	defNics := netutils2.SNicInfoList{}
	// fill in ovn vpc nic bridge field
	for _, nic := range s.GetDesc().Nics {
		if nic.Bridge == "" {
			nic.Bridge = getNicBridge(nic)
		}
		if nic.IsDefault {
			defaultGwCnt++
		}
		defNics = defNics.Add(nic.Ip, nic.Mac, nic.Gateway)
	}

	// there should 1 and only 1 default gateway
	if defaultGwCnt != 1 {
		// fix is_default
		_, defIndex := defNics.FindDefaultNicMac()
		for i := range s.GetDesc().Nics {
			if i == defIndex {
				s.GetDesc().Nics[i].IsDefault = true
			} else {
				s.GetDesc().Nics[i].IsDefault = false
			}
		}
	}

	if err := fileutils2.FilePutContents(
		s.GetDescFilePath(), jsonutils.Marshal(s.GetDesc()).String(), false,
	); err != nil {
		log.Errorf("save desc failed %s", err)
		return errors.Wrap(err, "save desc")
	}
	return nil
}

func GetPowerStates(s GuestRuntimeInstance) string {
	if s.IsRunning() {
		return computeapi.VM_POWER_STATES_ON
	} else {
		return computeapi.VM_POWER_STATES_OFF
	}
}

func DeleteHomeDir(s GuestRuntimeInstance) error {
	output, err := procutils.NewCommand("rm", "-rf", s.HomeDir()).Output()
	if err != nil {
		return errors.Wrapf(err, "rm %s failed: %s", s.HomeDir(), output)
	}
	return nil
}
