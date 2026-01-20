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
	"sync"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
)

var HostPendingUsageManager *SHostPendingUsageManager

type SHostPendingUsageManager struct {
	store              *SHostMemoryPendingUsageStore
	reloadAllStartTime time.Time    // 记录 ReloadAll 开始时间
	reloadStartTime    time.Time    // 记录部分 Reload 开始时间
	reloadAllLock      sync.RWMutex // 保护 reloadAllStartTime
	reloadLock         sync.RWMutex // 保护 reloadStartTime
}

func init() {
	pendingStore := NewHostMemoryPendingUsageStore()

	HostPendingUsageManager = &SHostPendingUsageManager{
		store:              pendingStore,
		reloadAllStartTime: time.Time{}, // 初始化为零值
		reloadStartTime:    time.Time{}, // 初始化为零值
	}
}

func (m *SHostPendingUsageManager) Keyword() string {
	return "pending_usage_manager"
}

func (m *SHostPendingUsageManager) newSessionUsage(req *api.SchedInfo, hostId string) *SessionPendingUsage {
	su := NewSessionUsage(req.SessionId, hostId)
	su.Usage = NewPendingUsageBySchedInfo(hostId, req, nil)
	// CreatedAt is already set in NewSessionUsage
	return su
}

func (m *SHostPendingUsageManager) newPendingUsage(hostId string, candidate *schedapi.CandidateResource) *SPendingUsage {
	return NewPendingUsageBySchedInfo(hostId, nil, candidate)
}

func (m *SHostPendingUsageManager) GetPendingUsage(hostId string) (*SPendingUsage, error) {
	return m.getPendingUsage(hostId)
}

func (m *SHostPendingUsageManager) GetNetPendingUsage(netId string) int {
	return m.store.GetNetPendingUsage(netId)
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

func (m *SHostPendingUsageManager) AddPendingUsage(req *api.SchedInfo, candidate *schedapi.CandidateResource) {
	hostId := candidate.HostId
	log.Infof("[PendingUsage] AddPendingUsage: sessionId=%s, hostId=%s, memory=%dMB, cpu=%d",
		req.SessionId, hostId, req.Memory, req.Ncpu)

	sessionUsage, _ := m.GetSessionUsage(req.SessionId, hostId)
	if sessionUsage == nil {
		sessionUsage = m.newSessionUsage(req, hostId)
		log.Infof("[PendingUsage] Created new SessionPendingUsage: %s", sessionUsage)
	}
	m.addSessionUsage(candidate.HostId, candidate, sessionUsage)
	if candidate.BackupCandidate != nil {
		m.AddPendingUsage(req, candidate.BackupCandidate)
	}
}

// addSessionUsage add pending usage and session usage
func (m *SHostPendingUsageManager) addSessionUsage(hostId string, candidate *schedapi.CandidateResource, usage *SessionPendingUsage) {
	ctx := context.Background()
	lockman.LockClass(ctx, m, hostId)
	defer lockman.ReleaseClass(ctx, m, hostId)

	pendingUsage, _ := m.getPendingUsage(hostId)
	if pendingUsage == nil {
		pendingUsage = m.newPendingUsage(hostId, candidate)
	}
	pendingUsage.Add(usage.Usage)
	usage.AddCount()
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

	oldMemory := pendingUsage.Memory
	oldCpu := pendingUsage.Cpu
	pendingUsage.Sub(su.Usage)
	m.store.SetPendingUsage(hostId, pendingUsage)
	su.SubCount()

	log.Infof("[PendingUsage] CancelPendingUsage: %s, host %s pending usage memory: %d->%dMB, cpu: %d->%d",
		su, hostId, oldMemory, pendingUsage.Memory, oldCpu, pendingUsage.Cpu)
	return nil
}

func (m *SHostPendingUsageManager) DeleteSessionUsage(usage *SessionPendingUsage) {
	m.store.DeleteSessionUsage(usage)
}

// GCExpiredSessionUsages releases session pending usage that has lived longer than ttl.
// This is a safety net to avoid leaked pending usages.
func (m *SHostPendingUsageManager) GCExpiredSessionUsages(ttl time.Duration) int {
	if ttl <= 0 {
		return 0
	}
	now := time.Now()
	expired := make([]*SessionPendingUsage, 0)

	m.store.RangeSessionUsages(func(su *SessionPendingUsage) bool {
		if su == nil {
			return true
		}
		if now.Sub(su.CreatedAt) > ttl {
			expired = append(expired, su)
		}
		return true
	})

	cleared := 0
	for _, su := range expired {
		hostId := su.Usage.HostId
		// best-effort cancel + delete
		_ = m.CancelPendingUsage(hostId, su)
		m.DeleteSessionUsage(su)
		cleared++
		log.Warningf("[PendingUsage] GCExpiredSessionUsage cleared: ttl=%v, now=%v, %s", ttl, now, su)
	}
	return cleared
}

// SetReloadStartTime marks the start of a partial reload operation
// This should be called before Reload to protect pending usage added during reload
func (m *SHostPendingUsageManager) SetReloadStartTime() {
	m.reloadLock.Lock()
	defer m.reloadLock.Unlock()
	m.reloadStartTime = time.Now()
	log.Infof("[PendingUsage] SetReloadStartTime: cutoff time set to %v", m.reloadStartTime)
}

// GetReloadStartTime returns the cutoff time for partial reload
func (m *SHostPendingUsageManager) GetReloadStartTime() time.Time {
	m.reloadLock.RLock()
	defer m.reloadLock.RUnlock()
	return m.reloadStartTime
}

// GetStore returns the underlying store for direct access
func (m *SHostPendingUsageManager) GetStore() *SHostMemoryPendingUsageStore {
	return m.store
}

// SetReloadAllStartTime marks the start of a full reload operation
// This should be called before ReloadAll to protect pending usage added during reload
func (m *SHostPendingUsageManager) SetReloadAllStartTime() {
	m.reloadAllLock.Lock()
	defer m.reloadAllLock.Unlock()
	m.reloadAllStartTime = time.Now()
	log.Infof("[PendingUsage] SetReloadAllStartTime: cutoff time set to %v", m.reloadAllStartTime)
}

// ClearAllPendingUsage clears all pending usage created before the last ReloadAll start
// This is called when all hosts are fully reloaded
func (m *SHostPendingUsageManager) ClearAllPendingUsage() {
	m.reloadAllLock.RLock()
	cutoffTime := m.reloadAllStartTime
	m.reloadAllLock.RUnlock()

	if cutoffTime.IsZero() {
		// No ReloadAll has been started, clear all
		log.Warningf("[PendingUsage] ClearAllPendingUsage: skipping clear all (no cutoff time)")
	} else {
		// Only clear pending usage created before ReloadAll started
		log.Infof("[PendingUsage] ClearAllPendingUsage: clearing created before %v", cutoffTime)
		m.store.clearAllPendingUsageBefore(cutoffTime)
		log.Infof("[PendingUsage] Cleared pending usage created before %v", cutoffTime)
	}
}

type SHostMemoryPendingUsageStore struct {
	store *sync.Map
}

func NewHostMemoryPendingUsageStore() *SHostMemoryPendingUsageStore {
	return &SHostMemoryPendingUsageStore{
		store: new(sync.Map),
	}
}

func (s *SHostMemoryPendingUsageStore) RangeSessionUsages(f func(*SessionPendingUsage) bool) {
	s.store.Range(func(_, v interface{}) bool {
		su, ok := v.(*SessionPendingUsage)
		if !ok || su == nil {
			return true
		}
		return f(su)
	})
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

func (self *SHostMemoryPendingUsageStore) GetNetPendingUsage(id string) int {
	total := 0
	self.store.Range(func(hostId, usageObj interface{}) bool {
		usage, ok := usageObj.(*SPendingUsage)
		if ok {
			total += usage.NetUsage.Get(id)
		}
		return true
	})
	return total
}

// clearPendingUsageBefore is the common implementation for clearing pending usage based on cutoffTime
func (self *SHostMemoryPendingUsageStore) clearPendingUsageBefore(
	shouldDelete func(*SessionPendingUsage) bool,
	logPrefix string,
	cutoffTime time.Time,
) {
	sessionKeysToDelete := make([]string, 0)
	hostIdsToDelete := make(map[string]bool)
	sessionUsagesToDelete := make(map[string]*SessionPendingUsage) // key -> sessionUsage

	// First pass: collect session usages to delete
	self.store.Range(func(key, value interface{}) bool {
		keyStr, ok := key.(string)
		if !ok {
			return true
		}

		// Check if it's a session usage
		if su, ok := value.(*SessionPendingUsage); ok {
			if shouldDelete(su) {
				sessionKeysToDelete = append(sessionKeysToDelete, keyStr)
				hostIdsToDelete[su.Usage.HostId] = true
				sessionUsagesToDelete[keyStr] = su
			}
		}
		return true
	})

	log.Infof("[PendingUsage] %s: found %d session usages to delete (cutoff=%v)",
		logPrefix, len(sessionKeysToDelete), cutoffTime)

	// Delete session usages and update pending usage
	deletedCount := 0
	for _, key := range sessionKeysToDelete {
		su := sessionUsagesToDelete[key]
		if su != nil {
			hostId := su.Usage.HostId
			// Update pending usage by subtracting this session usage
			if pendingUsage, err := self.GetPendingUsage(hostId); err == nil {
				oldMemory := pendingUsage.Memory
				oldCpu := pendingUsage.Cpu
				pendingUsage.Sub(su.Usage)
				if pendingUsage.IsEmpty() {
					self.store.Delete(hostId)
					log.Infof("[PendingUsage] Deleted empty pending usage for host %s", hostId)
				} else {
					self.store.Store(hostId, pendingUsage)
					log.Debugf("[PendingUsage] Updated pending usage for host %s: memory %d->%dMB, cpu %d->%d",
						hostId, oldMemory, pendingUsage.Memory, oldCpu, pendingUsage.Cpu)
				}
			}
		}
		// Delete session usage
		self.store.Delete(key)
		deletedCount++
	}
}

// ClearHostPendingUsageBefore clears pending usage for specified hosts created before cutoffTime
func (self *SHostMemoryPendingUsageStore) ClearHostPendingUsageBefore(hostIds []string, cutoffTime time.Time) {
	hostIdSet := make(map[string]bool)
	for _, hostId := range hostIds {
		hostIdSet[hostId] = true
	}

	self.clearPendingUsageBefore(
		func(su *SessionPendingUsage) bool {
			return hostIdSet[su.Usage.HostId] && su.CreatedAt.Before(cutoffTime)
		},
		fmt.Sprintf("ClearHostPendingUsageBefore: hosts %v", hostIds),
		cutoffTime,
	)
}

// clearAllPendingUsageBefore clears pending usage and session usages created before cutoffTime
func (self *SHostMemoryPendingUsageStore) clearAllPendingUsageBefore(cutoffTime time.Time) {
	self.clearPendingUsageBefore(
		func(su *SessionPendingUsage) bool {
			return su.CreatedAt.Before(cutoffTime)
		},
		"clearAllPendingUsageBefore",
		cutoffTime,
	)
}

type SessionPendingUsage struct {
	HostId    string
	SessionId string
	Usage     *SPendingUsage
	countLock *sync.Mutex
	count     int
	CreatedAt time.Time // 记录创建时间，用于 ReloadAll 时判断是否应该清空
}

func NewSessionUsage(sid, hostId string) *SessionPendingUsage {
	su := &SessionPendingUsage{
		HostId:    hostId,
		SessionId: sid,
		Usage:     NewPendingUsageBySchedInfo(hostId, nil, nil),
		count:     0,
		countLock: new(sync.Mutex),
		CreatedAt: time.Now(), // 记录创建时间
	}
	return su
}

func (su *SessionPendingUsage) GetHostId() string {
	return su.Usage.HostId
}

func (su *SessionPendingUsage) AddCount() {
	su.countLock.Lock()
	defer su.countLock.Unlock()
	su.count++
}

func (su *SessionPendingUsage) SubCount() {
	su.countLock.Lock()
	defer su.countLock.Unlock()
	su.count--
}

func (su *SessionPendingUsage) String() string {
	if su == nil {
		return "<nil>"
	}
	return fmt.Sprintf("SessionPendingUsage{sessionId=%s, hostId=%s, count=%d, createdAt=%v}: %s",
		su.SessionId, su.HostId, su.count, su.CreatedAt, jsonutils.Marshal(su.Usage.ToMap()))
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
	HostId         string
	Cpu            int
	Memory         int
	IsolatedDevice int
	DiskUsage      *SResourcePendingUsage
	NetUsage       *SResourcePendingUsage
	// Lock is not need here
	InstanceGroupUsage map[string]*api.CandidateGroup
}

func NewPendingUsageBySchedInfo(hostId string, req *api.SchedInfo, candidate *schedapi.CandidateResource) *SPendingUsage {
	u := &SPendingUsage{
		HostId:    hostId,
		DiskUsage: NewResourcePendingUsage(nil),
		NetUsage:  NewResourcePendingUsage(nil),
	}

	// group init
	u.InstanceGroupUsage = make(map[string]*api.CandidateGroup)

	if req == nil {
		return u
	}
	u.Cpu = req.Ncpu
	u.Memory = req.Memory
	u.IsolatedDevice = len(req.IsolatedDevices)

	for _, disk := range req.Disks {
		backend := disk.Backend
		size := disk.SizeMb
		osize := u.DiskUsage.Get(backend)
		u.DiskUsage.Set(backend, osize+size)
	}

	if candidate != nil && len(candidate.Nets) > 0 {
		for _, net := range candidate.Nets {
			// 只对建议 network_id 为1个的时候设置 pending_usage
			// 多个的情况下只有交给 region 那边自己判断
			// 这里只是尽让调度器提前判断出子网是否空闲 ip
			if len(net.NetworkIds) != 1 {
				continue
			}
			id := net.NetworkIds[0]
			ocount := u.NetUsage.Get(id)
			u.NetUsage.Set(id, ocount+1)
		}
	} else {
		for _, net := range req.Networks {
			id := net.Network
			if id == "" {
				continue
			}
			ocount := u.NetUsage.Get(id)
			u.NetUsage.Set(id, ocount+1)
		}

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

func (self *SPendingUsage) Add(sUsage *SPendingUsage) {
	self.Cpu = self.Cpu + sUsage.Cpu
	self.Memory = self.Memory + sUsage.Memory
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
	self.Memory = quotas.NonNegative(self.Memory - sUsage.Memory)
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
