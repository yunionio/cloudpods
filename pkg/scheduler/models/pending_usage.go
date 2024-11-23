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

package models

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
)

var HostPendingUsageManager *SHostPendingUsageManager

type SHostPendingUsageManager struct {
	store *SHostMemoryPendingUsageStore
}

func init() {
	pendingStore := NewHostMemoryPendingUsageStore()

	HostPendingUsageManager = &SHostPendingUsageManager{
		store: pendingStore,
	}
}

func (m *SHostPendingUsageManager) Keyword() string {
	return "pending_usage_manager"
}

func (m *SHostPendingUsageManager) newSessionUsage(req *api.SchedInfo, hostId string, candidate *schedapi.CandidateResource) *SessionPendingUsage {
	usage := NewPendingUsageBySchedInfo(hostId, req, candidate)
	su := NewSessionUsage(req.SessionId, hostId, usage)
	return su
}

func (m *SHostPendingUsageManager) GetPendingUsage(hostId string) (*SPendingUsage, error) {
	return m.getPendingUsage(hostId)
}

func (m *SHostPendingUsageManager) getPendingUsage(hostId string) (*SPendingUsage, error) {
	pending, err := m.store.GetPendingUsage(hostId)
	if err != nil {
		return nil, err
	}
	return pending, nil
}

func (m *SHostPendingUsageManager) GetSessionUsage(sessionId, hostId string) (*SessionPendingUsage, error) {
	return m.store.GetSessionUsage(sessionId, hostId)
}

func (m *SHostPendingUsageManager) AddPendingUsage(guestId string, req *api.SchedInfo, candidate *schedapi.CandidateResource) {
	hostId := candidate.HostId

	sessionUsage, _ := m.GetSessionUsage(req.SessionId, hostId)
	if sessionUsage == nil {
		sessionUsage = m.newSessionUsage(req, hostId, candidate)
		sessionUsage.StartTimer()
	}
	m.addSessionUsage(candidate.HostId, guestId, sessionUsage)
	if candidate.BackupCandidate != nil {
		m.AddPendingUsage(guestId, req, candidate.BackupCandidate)
	}
}

// addSessionUsage add pending usage and session usage
func (m *SHostPendingUsageManager) addSessionUsage(hostId, guestId string, usage *SessionPendingUsage) {
	ctx := context.Background()
	lockman.LockClass(ctx, m, hostId)
	defer lockman.ReleaseClass(ctx, m, hostId)

	pendingUsage, _ := m.getPendingUsage(hostId)
	if pendingUsage == nil {
		pendingUsage = NewPendingUsageBySchedInfo(hostId, nil, nil)
	}
	// add pending usage
	pendingUsage.Add(usage.Usage, guestId)
	usage.AddCount(guestId)
	m.store.SetSessionUsage(usage.SessionId, hostId, usage)
	m.store.SetPendingUsage(hostId, pendingUsage)
}

func (m *SHostPendingUsageManager) CancelPendingUsage(hostId string, su *SessionPendingUsage) error {
	ctx := context.Background()
	lockman.LockClass(ctx, m, hostId)
	defer lockman.ReleaseClass(ctx, m, hostId)

	pendingUsage, _ := m.getPendingUsage(hostId)
	if pendingUsage == nil {
		return nil
	}
	if su == nil {
		return nil
	}
	pendingUsage.Sub(su.Usage)
	m.store.SetPendingUsage(hostId, pendingUsage)
	su.SubCount()
	return nil
}

func (m *SHostPendingUsageManager) DeleteSessionUsage(usage *SessionPendingUsage) {
	m.store.DeleteSessionUsage(usage)
}

type SHostMemoryPendingUsageStore struct {
	store *sync.Map
}

func NewHostMemoryPendingUsageStore() *SHostMemoryPendingUsageStore {
	return &SHostMemoryPendingUsageStore{
		store: new(sync.Map),
	}
}

func (self *SHostMemoryPendingUsageStore) sessionUsageKey(sid, hostId string) string {
	return fmt.Sprintf("%s-%s", sid, hostId)
}

func (self *SHostMemoryPendingUsageStore) GetSessionUsage(sessionId string, hostId string) (*SessionPendingUsage, error) {
	key := self.sessionUsageKey(sessionId, hostId)
	ret, ok := self.store.Load(key)
	if !ok {
		return nil, errors.Errorf("Not fond session pending usage by %s", key)
	}
	return ret.(*SessionPendingUsage), nil
}

func (self *SHostMemoryPendingUsageStore) SetSessionUsage(sessionId, hostId string, usage *SessionPendingUsage) {
	key := self.sessionUsageKey(sessionId, hostId)
	self.store.Store(key, usage)
}

func (self *SHostMemoryPendingUsageStore) GetPendingUsage(hostId string) (*SPendingUsage, error) {
	ret, ok := self.store.Load(hostId)
	if !ok {
		return nil, errors.Errorf("Not fond pending usage by %s", hostId)
	}
	usage := ret.(*SPendingUsage)
	return usage, nil
}

func (self *SHostMemoryPendingUsageStore) SetPendingUsage(hostId string, usage *SPendingUsage) {
	if usage.IsEmpty() {
		self.store.Delete(hostId)
		return
	}
	self.store.Store(hostId, usage)
}

func (self *SHostMemoryPendingUsageStore) DeleteSessionUsage(usage *SessionPendingUsage) {
	self.store.Delete(self.sessionUsageKey(usage.SessionId, usage.Usage.HostId))
}

type SessionPendingUsage struct {
	HostId    string
	SessionId string
	Usage     *SPendingUsage
	countLock *sync.Mutex
	count     int
	cancelCh  chan string
}

func NewSessionUsage(sid, hostId string, usage *SPendingUsage) *SessionPendingUsage {
	su := &SessionPendingUsage{
		HostId:    hostId,
		SessionId: sid,
		Usage:     usage,
		count:     0,
		countLock: new(sync.Mutex),
		cancelCh:  make(chan string),
	}
	return su
}

func (su *SessionPendingUsage) GetHostId() string {
	return su.Usage.HostId
}

func (su *SessionPendingUsage) AddCount(guestId string) {
	su.countLock.Lock()
	defer su.countLock.Unlock()
	su.count++
	su.Usage.PendingGuestIds[guestId] = struct{}{}
}

func (su *SessionPendingUsage) SubCount() {
	su.countLock.Lock()
	defer su.countLock.Unlock()
	su.count--
	for guestId, _ := range su.Usage.PendingGuestIds {
		delete(su.Usage.PendingGuestIds, guestId)
		break
	}
}

type SResourcePendingUsage struct {
	store *sync.Map
}

func NewResourcePendingUsage(vals map[string]int) *SResourcePendingUsage {
	u := &SResourcePendingUsage{
		store: new(sync.Map),
	}
	for key, val := range vals {
		u.Set(key, val)
	}
	return u
}

func (u *SResourcePendingUsage) ToMap() map[string]int {
	ret := make(map[string]int)
	u.Range(func(key string, val int) bool {
		ret[key] = val
		return true
	})
	return ret
}

func (u *SResourcePendingUsage) Get(key string) int {
	val, ok := u.store.Load(key)
	if !ok {
		return 0
	}
	return val.(int)
}

func (u *SResourcePendingUsage) Set(key string, size int) {
	u.store.Store(key, size)
}

func (u *SResourcePendingUsage) Range(f func(key string, size int) bool) {
	u.store.Range(func(key, val interface{}) bool {
		return f(key.(string), val.(int))
	})
}

func (u *SResourcePendingUsage) Add(su *SResourcePendingUsage) {
	u.Range(func(key string, size int) bool {
		size2 := su.Get(key)
		u.Set(key, size+size2)
		return true
	})
	su.Range(func(key string, size int) bool {
		if _, ok := u.store.Load(key); !ok {
			u.Set(key, size)
		}
		return true
	})
}

func (u *SResourcePendingUsage) Sub(su *SResourcePendingUsage) {
	u.Range(func(key string, size int) bool {
		size2 := su.Get(key)
		u.Set(key, quotas.NonNegative(size-size2))
		return true
	})
}

func (u *SResourcePendingUsage) IsEmpty() bool {
	empty := true
	u.Range(func(_ string, size int) bool {
		if size != 0 {
			empty = false
			return false
		}
		return true
	})
	return empty
}

type SPendingUsage struct {
	HostId string
	Cpu    int
	CpuPin map[int]int
	Memory int

	PendingGuestIds map[string]struct{}

	// nodeId: memSizeMB
	NumaMemPin     map[int]int
	IsolatedDevice int
	DiskUsage      *SResourcePendingUsage
	NetUsage       *SResourcePendingUsage
	// Lock is not need here
	InstanceGroupUsage map[string]*api.CandidateGroup
}

func NewPendingUsageBySchedInfo(hostId string, req *api.SchedInfo, candidate *schedapi.CandidateResource) *SPendingUsage {
	u := &SPendingUsage{
		HostId:          hostId,
		DiskUsage:       NewResourcePendingUsage(nil),
		NetUsage:        NewResourcePendingUsage(nil),
		PendingGuestIds: make(map[string]struct{}),
	}

	// group init
	u.InstanceGroupUsage = make(map[string]*api.CandidateGroup)
	u.CpuPin = make(map[int]int)
	u.NumaMemPin = make(map[int]int)

	if req == nil {
		return u
	}
	u.Cpu = req.Ncpu
	u.Memory = req.Memory
	u.IsolatedDevice = len(req.IsolatedDevices)

	if candidate != nil && len(candidate.CpuNumaPin) > 0 {
		for _, cpuNumaPin := range candidate.CpuNumaPin {
			if cpuNumaPin.MemSizeMB != nil {
				if v, ok := u.NumaMemPin[cpuNumaPin.NodeId]; ok {
					u.NumaMemPin[cpuNumaPin.NodeId] = v + *cpuNumaPin.MemSizeMB
				} else {
					u.NumaMemPin[cpuNumaPin.NodeId] = *cpuNumaPin.MemSizeMB
				}
			}

			for i := range cpuNumaPin.CpuPin {
				if v, ok := u.CpuPin[cpuNumaPin.CpuPin[i]]; ok {
					u.CpuPin[cpuNumaPin.CpuPin[i]] = v + 1
				} else {
					u.CpuPin[cpuNumaPin.CpuPin[i]] = 1
				}
			}
		}
	}

	for _, disk := range req.Disks {
		backend := disk.Backend
		size := disk.SizeMb
		osize := u.DiskUsage.Get(backend)
		u.DiskUsage.Set(backend, osize+size)
	}

	for _, net := range req.Networks {
		id := net.Network
		if id == "" {
			continue
		}
		ocount := u.NetUsage.Get(id)
		u.NetUsage.Set(id, ocount+1)
	}

	// group add
	for _, groupId := range req.InstanceGroupIds {
		// For now, info about instancegroup in api.SchedInfo is only "ID",
		// but in the future, info may increase
		group := &computemodels.SGroup{}
		group.Id = groupId
		u.InstanceGroupUsage[groupId] = &api.CandidateGroup{
			SGroup:     group,
			ReferCount: 1,
		}
	}

	return u
}

func (self *SPendingUsage) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"cpu":             self.Cpu,
		"memory":          self.Memory,
		"isolated_device": self.IsolatedDevice,
		"disk":            self.DiskUsage.ToMap(),
		"net":             self.NetUsage.ToMap(),
		"instance_groups": self.InstanceGroupUsage,
	}
}

func (self *SPendingUsage) Add(sUsage *SPendingUsage, addGuestId string) {
	self.Cpu = self.Cpu + sUsage.Cpu
	for k, v1 := range sUsage.CpuPin {
		if v2, ok := self.CpuPin[k]; ok {
			self.CpuPin[k] = v1 + v2
		} else {
			self.CpuPin[k] = v1
		}
	}

	for guestId := range sUsage.PendingGuestIds {
		if _, ok := self.PendingGuestIds[guestId]; !ok {
			log.Infof("add guest %s in pending usage", guestId)
			self.PendingGuestIds[guestId] = struct{}{}
		}
	}
	if addGuestId != "" {
		log.Infof("add guest %s in pending usage", addGuestId)
		self.PendingGuestIds[addGuestId] = struct{}{}
	}

	self.Memory = self.Memory + sUsage.Memory
	for k, v1 := range sUsage.NumaMemPin {
		if v2, ok := self.NumaMemPin[k]; ok {
			self.NumaMemPin[k] = v1 + v2
		} else {
			self.NumaMemPin[k] = v1
		}
	}
	self.IsolatedDevice = self.IsolatedDevice + sUsage.IsolatedDevice
	self.DiskUsage.Add(sUsage.DiskUsage)
	self.NetUsage.Add(sUsage.NetUsage)
	for id, cg := range sUsage.InstanceGroupUsage {
		if scg, ok := self.InstanceGroupUsage[id]; ok {
			scg.ReferCount += cg.ReferCount
			continue
		}
		self.InstanceGroupUsage[id] = cg
	}
}

func (self *SPendingUsage) Sub(sUsage *SPendingUsage) {
	self.Cpu = quotas.NonNegative(self.Cpu - sUsage.Cpu)
	for k, v1 := range sUsage.CpuPin {
		if v2, ok := self.CpuPin[k]; ok {
			self.CpuPin[k] = quotas.NonNegative(v2 - v1)
		}
	}

	for guestId := range sUsage.PendingGuestIds {
		log.Infof("delete guest %s in pending usage", guestId)
		delete(self.PendingGuestIds, guestId)
	}

	self.Memory = quotas.NonNegative(self.Memory - sUsage.Memory)
	for k, v1 := range sUsage.NumaMemPin {
		if v2, ok := self.NumaMemPin[k]; ok {
			self.NumaMemPin[k] = quotas.NonNegative(v2 - v1)
		}
	}

	self.IsolatedDevice = quotas.NonNegative(self.IsolatedDevice - sUsage.IsolatedDevice)
	self.DiskUsage.Sub(sUsage.DiskUsage)
	self.NetUsage.Sub(sUsage.NetUsage)
	for id, cg := range sUsage.InstanceGroupUsage {
		if scg, ok := self.InstanceGroupUsage[id]; ok {
			count := scg.ReferCount - cg.ReferCount
			if count <= 0 {
				delete(self.InstanceGroupUsage, id)
				continue
			}
			scg.ReferCount = count
		}
	}
}

func (self *SPendingUsage) IsEmpty() bool {
	if self.Cpu > 0 {
		return false
	}
	if self.Memory > 0 {
		return false
	}
	if self.IsolatedDevice > 0 {
		return false
	}
	if !self.DiskUsage.IsEmpty() {
		return false
	}
	if !self.NetUsage.IsEmpty() {
		return false
	}
	if len(self.InstanceGroupUsage) != 0 {
		return false
	}
	return true
}

func (self *SessionPendingUsage) cancelSelf() {
	hostId := self.Usage.HostId
	count := self.count

	for i := 0; i <= count; i++ {
		HostPendingUsageManager.CancelPendingUsage(hostId, self)
	}
}

func (self *SessionPendingUsage) StartTimer() {
	timeout := 1 * time.Minute
	go func() {
		for {
			select {
			case <-time.After(timeout):
				log.Infof("timeout cancel session usage %#v", self)
				self.cancelSelf()
				goto ForEnd
			case sid := <-self.cancelCh:
				log.Infof("Cancel session %s usage, count: %d", sid, self.count)
				if self.count <= 0 {
					goto ForEnd
				} else {
					log.Infof("continue waiting next cancel...")
				}
			}
		}
	ForEnd:
		log.Infof("delete session usage %#v", self)
		HostPendingUsageManager.DeleteSessionUsage(self)
	}()
}

func (self *SessionPendingUsage) StopTimer() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("SessionPendingUsage %#v stop timer: %v", self, r)
			debug.PrintStack()
		}
	}()
	self.cancelCh <- self.SessionId
}
