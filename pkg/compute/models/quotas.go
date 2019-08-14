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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SQuotaManager struct {
	quotas.SQuotaBaseManager
}

var QuotaManager *SQuotaManager
var QuotaUsageManager *SQuotaManager

func init() {
	pendingStore := quotas.NewMemoryQuotaStore()

	QuotaUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(SQuota{}, "quota_usage_tbl"),
	}

	QuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(SQuota{}, "quota_tbl", pendingStore, QuotaUsageManager),
	}
}

type SQuota struct {
	quotas.SQuotaBase

	Cpu     int
	Memory  int
	Storage int
	Port    int
	Eip     int
	Eport   int
	Bw      int
	Ebw     int
	// Keypair int
	// Image          int
	Group          int
	Secgroup       int
	IsolatedDevice int
	Snapshot       int

	Bucket    int
	ObjectGB  int
	ObjectCnt int
}

func (self *SQuota) FetchSystemQuota(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) {
	base := 0
	if scope == rbacutils.ScopeDomain {
		base = 10
	} else if ownerId.GetProjectDomainId() == identityapi.DEFAULT_DOMAIN_ID && ownerId.GetProjectName() == identityapi.SystemAdminProject {
		base = 1
	}
	self.Cpu = options.Options.DefaultCpuQuota * base
	self.Memory = options.Options.DefaultMemoryQuota * base
	self.Storage = options.Options.DefaultStorageQuota * base
	self.Port = options.Options.DefaultPortQuota * base
	self.Eip = options.Options.DefaultEipQuota * base
	self.Eport = options.Options.DefaultEportQuota * base
	self.Bw = options.Options.DefaultBwQuota * base
	self.Ebw = options.Options.DefaultEbwQuota * base
	self.Group = options.Options.DefaultGroupQuota * base
	self.Secgroup = options.Options.DefaultSecgroupQuota * base
	self.IsolatedDevice = options.Options.DefaultIsolatedDeviceQuota * base
	self.Snapshot = options.Options.DefaultSnapshotQuota * base
	self.Bucket = options.Options.DefaultBucketQuota * base
	self.ObjectGB = options.Options.DefaultObjectGBQuota * base
	self.ObjectCnt = options.Options.DefaultObjectCntQuota * base
}

func (self *SQuota) FetchUsage(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, name []string) error {
	diskSize := totalDiskSize(scope, ownerId, tristate.None, tristate.None, false, false)
	net := totalGuestNicCount(scope, ownerId, nil, false)
	lbnic, _ := totalLBNicCount(scope, ownerId)
	// net := WireManager.TotalCount(nil, nil, nil, "", scope, ownerId)
	hypervisors := sets.NewString(api.HYPERVISORS...)
	hypervisors.Delete(api.HYPERVISOR_CONTAINER)
	guest := totalGuestResourceCount(scope, ownerId, nil, nil, hypervisors.List(), false, false, nil, nil, nil, nil, "")
	eipUsage := ElasticipManager.TotalCount(scope, ownerId, nil, nil, nil, "")
	snapshotCount, _ := TotalSnapshotCount(scope, ownerId, nil, nil, nil, "")
	bucketUsage := BucketManager.TotalCount(scope, ownerId, nil, nil, nil, "")
	// XXX
	// keypair belongs to user
	// keypair := totalKeypairCount(projectId)

	self.Cpu = guest.TotalCpuCount
	self.Memory = guest.TotalMemSize
	self.Storage = diskSize
	self.Eip = eipUsage.Total()
	// log.Debugf("%d %d %d\n", net.InternalNicCount, net.InternalVirtualNicCount, lbnic)
	self.Port = net.InternalNicCount + net.InternalVirtualNicCount + lbnic
	self.Eport = net.ExternalNicCount + net.ExternalVirtualNicCount
	self.Bw = net.InternalBandwidth
	self.Ebw = net.ExternalBandwidth
	// self.Port = net.GuestNicCount + net.GroupNicCount + net.LbNicCount
	// if scope == rbacutils.ScopeSystem {
	// 	self.Port += net.HostNicCount + net.ReservedCount
	// }
	self.Group = 0
	self.Secgroup, _ = totalSecurityGroupCount(scope, ownerId)
	self.IsolatedDevice = guest.TotalIsolatedCount
	self.Snapshot = snapshotCount
	self.Bucket = bucketUsage.Buckets
	self.ObjectGB = int(bucketUsage.Bytes / 1000 / 1000 / 1000)
	self.ObjectCnt = bucketUsage.Objects
	return nil
}

func (self *SQuota) IsEmpty() bool {
	if self.Cpu > 0 {
		return false
	}
	if self.Memory > 0 {
		return false
	}
	if self.Storage > 0 {
		return false
	}
	if self.Port > 0 {
		return false
	}
	if self.Eip > 0 {
		return false
	}
	if self.Eport > 0 {
		return false
	}
	if self.Bw > 0 {
		return false
	}
	if self.Ebw > 0 {
		return false
	}
	if self.Group > 0 {
		return false
	}
	if self.Secgroup > 0 {
		return false
	}
	if self.IsolatedDevice > 0 {
		return false
	}
	if self.Snapshot > 0 {
		return false
	}
	if self.Bucket > 0 {
		return false
	}
	if self.ObjectGB > 0 {
		return false
	}
	if self.ObjectCnt > 0 {
		return false
	}
	return true
}

func (self *SQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	self.Cpu = self.Cpu + squota.Cpu
	self.Memory = self.Memory + squota.Memory
	self.Storage = self.Storage + squota.Storage
	self.Port = self.Port + squota.Port
	self.Eip = self.Eip + squota.Eip
	self.Eport = self.Eport + squota.Eport
	self.Bw = self.Bw + squota.Bw
	self.Ebw = self.Ebw + squota.Ebw
	self.Group = self.Group + squota.Group
	self.Secgroup = self.Secgroup + squota.Secgroup
	self.IsolatedDevice = self.IsolatedDevice + squota.IsolatedDevice
	self.Snapshot = self.Snapshot + squota.Snapshot
	self.Bucket = self.Bucket + squota.Bucket
	self.ObjectGB = self.ObjectGB + squota.ObjectGB
	self.ObjectCnt = self.ObjectCnt + squota.ObjectCnt
}

func nonNegative(val int) int {
	return quotas.NonNegative(val)
}

func (self *SQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	self.Cpu = nonNegative(self.Cpu - squota.Cpu)
	self.Memory = nonNegative(self.Memory - squota.Memory)
	self.Storage = nonNegative(self.Storage - squota.Storage)
	self.Port = nonNegative(self.Port - squota.Port)
	self.Eip = nonNegative(self.Eip - squota.Eip)
	self.Eport = nonNegative(self.Eport - squota.Eport)
	self.Bw = nonNegative(self.Bw - squota.Bw)
	self.Ebw = nonNegative(self.Ebw - squota.Ebw)
	self.Group = nonNegative(self.Group - squota.Group)
	self.Secgroup = nonNegative(self.Secgroup - squota.Secgroup)
	self.IsolatedDevice = nonNegative(self.IsolatedDevice - squota.IsolatedDevice)
	self.Snapshot = nonNegative(self.Snapshot - squota.Snapshot)
	self.Bucket = nonNegative(self.Bucket - squota.Bucket)
	self.ObjectGB = nonNegative(self.ObjectGB - squota.ObjectGB)
	self.ObjectCnt = nonNegative(self.ObjectCnt - squota.ObjectCnt)
}

func (self *SQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	if squota.Cpu > 0 {
		self.Cpu = squota.Cpu
	}
	if squota.Memory > 0 {
		self.Memory = squota.Memory
	}
	if squota.Storage > 0 {
		self.Storage = squota.Storage
	}
	if squota.Port > 0 {
		self.Port = squota.Port
	}
	if squota.Eip > 0 {
		self.Eip = squota.Eip
	}
	if squota.Eport > 0 {
		self.Eport = squota.Eport
	}
	if squota.Bw > 0 {
		self.Bw = squota.Bw
	}
	if squota.Ebw > 0 {
		self.Ebw = squota.Ebw
	}
	if squota.Group > 0 {
		self.Group = squota.Group
	}
	if squota.Secgroup > 0 {
		self.Secgroup = squota.Secgroup
	}
	if squota.IsolatedDevice > 0 {
		self.IsolatedDevice = squota.IsolatedDevice
	}
	if squota.Snapshot > 0 {
		self.Snapshot = squota.Snapshot
	}
	if squota.Bucket > 0 {
		self.Bucket = squota.Bucket
	}
	if squota.ObjectGB > 0 {
		self.ObjectGB = squota.ObjectGB
	}
	if squota.ObjectCnt > 0 {
		self.ObjectCnt = squota.ObjectCnt
	}
}

func (self *SQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SQuota)
	squota := quota.(*SQuota)
	if sreq.Cpu > 0 && self.Cpu > squota.Cpu {
		err.Add("cpu", squota.Cpu, self.Cpu)
	}
	if sreq.Memory > 0 && self.Memory > squota.Memory {
		err.Add("memory", squota.Memory, self.Memory)
	}
	if sreq.Storage > 0 && self.Storage > squota.Storage {
		err.Add("storage", squota.Storage, self.Storage)
	}
	if sreq.Port > 0 && self.Port > squota.Port {
		err.Add("port", squota.Port, self.Port)
	}
	if sreq.Eip > 0 && self.Eip > squota.Eip {
		err.Add("eip", squota.Eip, self.Eip)
	}
	if sreq.Eport > 0 && self.Eport > squota.Eport {
		err.Add("eport", squota.Eport, self.Eport)
	}
	if sreq.Bw > 0 && self.Bw > squota.Bw {
		err.Add("bw", squota.Bw, self.Bw)
	}
	if sreq.Ebw > 0 && self.Ebw > squota.Ebw {
		err.Add("ebw", squota.Ebw, self.Ebw)
	}
	if sreq.Group > 0 && self.Group > squota.Group {
		err.Add("group", squota.Group, self.Group)
	}
	if sreq.Secgroup > 0 && self.Secgroup > squota.Secgroup {
		err.Add("secgroup", squota.Secgroup, self.Secgroup)
	}
	if sreq.IsolatedDevice > 0 && self.IsolatedDevice > squota.IsolatedDevice {
		err.Add("isolated_device", squota.IsolatedDevice, self.IsolatedDevice)
	}
	if sreq.Snapshot > 0 && self.Snapshot > squota.Snapshot {
		err.Add("snapshot", squota.Snapshot, self.Snapshot)
	}
	if sreq.Bucket > 0 && self.Bucket > squota.Bucket {
		err.Add("bucket", squota.Bucket, self.Bucket)
	}
	if sreq.ObjectGB > 0 && self.ObjectGB > squota.ObjectGB {
		err.Add("object_gb", squota.ObjectGB, self.ObjectGB)
	}
	if sreq.ObjectCnt > 0 && self.ObjectCnt > squota.ObjectCnt {
		err.Add("object_cnt", squota.ObjectCnt, self.ObjectCnt)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func keyName(prefix, name string) string {
	if len(prefix) > 0 {
		return fmt.Sprintf("%s.%s", prefix, name)
	} else {
		return name
	}
}

func (self *SQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(int64(self.Cpu)), keyName(prefix, "cpu"))
	ret.Add(jsonutils.NewInt(int64(self.Memory)), keyName(prefix, "memory"))
	ret.Add(jsonutils.NewInt(int64(self.Storage)), keyName(prefix, "storage"))
	ret.Add(jsonutils.NewInt(int64(self.Port)), keyName(prefix, "port"))
	ret.Add(jsonutils.NewInt(int64(self.Eip)), keyName(prefix, "eip"))
	ret.Add(jsonutils.NewInt(int64(self.Eport)), keyName(prefix, "eport"))
	ret.Add(jsonutils.NewInt(int64(self.Bw)), keyName(prefix, "bw"))
	ret.Add(jsonutils.NewInt(int64(self.Ebw)), keyName(prefix, "ebw"))
	ret.Add(jsonutils.NewInt(int64(self.Group)), keyName(prefix, "group"))
	ret.Add(jsonutils.NewInt(int64(self.Secgroup)), keyName(prefix, "secgroup"))
	ret.Add(jsonutils.NewInt(int64(self.IsolatedDevice)), keyName(prefix, "isolated_device"))
	ret.Add(jsonutils.NewInt(int64(self.Snapshot)), keyName(prefix, "snapshot"))
	ret.Add(jsonutils.NewInt(int64(self.Bucket)), keyName(prefix, "bucket"))
	ret.Add(jsonutils.NewInt(int64(self.ObjectGB)), keyName(prefix, "object_gb"))
	ret.Add(jsonutils.NewInt(int64(self.ObjectCnt)), keyName(prefix, "object_cnt"))
	return ret
}
