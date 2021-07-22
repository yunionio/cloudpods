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

package mock

import (
	"math"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	jsonutils "yunion.io/x/jsonutils"

	types "yunion.io/x/onecloud/pkg/cloudcommon/types"
	baremetal "yunion.io/x/onecloud/pkg/compute/baremetal"
	models "yunion.io/x/onecloud/pkg/compute/models"
	api "yunion.io/x/onecloud/pkg/scheduler/api"
	core "yunion.io/x/onecloud/pkg/scheduler/core"
	sku "yunion.io/x/onecloud/pkg/scheduler/data_manager/sku"
	models0 "yunion.io/x/onecloud/pkg/scheduler/models"
)

// MockCandidatePropertyGetter is a mock of CandidatePropertyGetter interface
type MockCandidatePropertyGetter struct {
	ctrl     *gomock.Controller
	recorder *MockCandidatePropertyGetterMockRecorder
}

// MockCandidatePropertyGetterMockRecorder is the mock recorder for MockCandidatePropertyGetter
type MockCandidatePropertyGetterMockRecorder struct {
	mock *MockCandidatePropertyGetter
}

// NewMockCandidatePropertyGetter creates a new mock instance
func NewMockCandidatePropertyGetter(ctrl *gomock.Controller) *MockCandidatePropertyGetter {
	mock := &MockCandidatePropertyGetter{ctrl: ctrl}
	mock.recorder = &MockCandidatePropertyGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCandidatePropertyGetter) EXPECT() *MockCandidatePropertyGetterMockRecorder {
	return m.recorder
}

// Cloudprovider mocks base method
func (m *MockCandidatePropertyGetter) Cloudprovider() *models.SCloudprovider {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cloudprovider")
	ret0, _ := ret[0].(*models.SCloudprovider)
	return ret0
}

// Cloudprovider indicates an expected call of Cloudprovider
func (mr *MockCandidatePropertyGetterMockRecorder) Cloudprovider() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cloudprovider", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Cloudprovider))
}

// CreatingGuestCount mocks base method
func (m *MockCandidatePropertyGetter) CreatingGuestCount() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatingGuestCount")
	ret0, _ := ret[0].(int)
	return ret0
}

// CreatingGuestCount indicates an expected call of CreatingGuestCount
func (mr *MockCandidatePropertyGetterMockRecorder) CreatingGuestCount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatingGuestCount", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).CreatingGuestCount))
}

// DomainId mocks base method
func (m *MockCandidatePropertyGetter) DomainId() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DomainId")
	ret0, _ := ret[0].(string)
	return ret0
}

// DomainId indicates an expected call of DomainId
func (mr *MockCandidatePropertyGetterMockRecorder) DomainId() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DomainId", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).DomainId))
}

// Enabled mocks base method
func (m *MockCandidatePropertyGetter) Enabled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enabled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Enabled indicates an expected call of Enabled
func (mr *MockCandidatePropertyGetterMockRecorder) Enabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enabled", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Enabled))
}

// FreeCPUCount mocks base method
func (m *MockCandidatePropertyGetter) FreeCPUCount(arg0 bool) int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FreeCPUCount", arg0)
	ret0, _ := ret[0].(int64)
	return ret0
}

// FreeCPUCount indicates an expected call of FreeCPUCount
func (mr *MockCandidatePropertyGetterMockRecorder) FreeCPUCount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeCPUCount", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).FreeCPUCount), arg0)
}

// FreeMemorySize mocks base method
func (m *MockCandidatePropertyGetter) FreeMemorySize(arg0 bool) int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FreeMemorySize", arg0)
	ret0, _ := ret[0].(int64)
	return ret0
}

// FreeMemorySize indicates an expected call of FreeMemorySize
func (mr *MockCandidatePropertyGetterMockRecorder) FreeMemorySize(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeMemorySize", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).FreeMemorySize), arg0)
}

// GetFreeGroupCount mocks base method
func (m *MockCandidatePropertyGetter) GetFreeGroupCount(arg0 string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFreeGroupCount", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFreeGroupCount indicates an expected call of GetFreeGroupCount
func (mr *MockCandidatePropertyGetterMockRecorder) GetFreeGroupCount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFreeGroupCount", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetFreeGroupCount), arg0)
}

// GetFreePort mocks base method
func (m *MockCandidatePropertyGetter) GetFreePort(arg0 string) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFreePort", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// GetFreePort indicates an expected call of GetFreePort
func (mr *MockCandidatePropertyGetterMockRecorder) GetFreePort(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFreePort", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetFreePort), arg0)
}

// GetFreeStorageSizeOfType mocks base method
func (m *MockCandidatePropertyGetter) GetFreeStorageSizeOfType(arg0 string, arg1 bool) (int64, int64) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFreeStorageSizeOfType", arg0, arg1)
	ret0, _ := ret[0].(int64)
	return ret0, math.MaxInt64
}

// GetFreeStorageSizeOfType indicates an expected call of GetFreeStorageSizeOfType
func (mr *MockCandidatePropertyGetterMockRecorder) GetFreeStorageSizeOfType(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFreeStorageSizeOfType", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetFreeStorageSizeOfType), arg0, arg1)
}

// GetIpmiInfo mocks base method
func (m *MockCandidatePropertyGetter) GetIpmiInfo() types.SIPMIInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIpmiInfo")
	ret0, _ := ret[0].(types.SIPMIInfo)
	return ret0
}

// GetIpmiInfo indicates an expected call of GetIpmiInfo
func (mr *MockCandidatePropertyGetterMockRecorder) GetIpmiInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIpmiInfo", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetIpmiInfo))
}

// GetNics mocks base method
func (m *MockCandidatePropertyGetter) GetNics() []*types.SNic {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNics")
	ret0, _ := ret[0].([]*types.SNic)
	return ret0
}

// GetIpmiInfo indicates an expected call of GetIpmiInfo
func (mr *MockCandidatePropertyGetterMockRecorder) GetNics() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNics", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetNics))
}

// GetIsolatedDevice mocks base method
func (m *MockCandidatePropertyGetter) GetIsolatedDevice(arg0 string) *core.IsolatedDeviceDesc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIsolatedDevice", arg0)
	ret0, _ := ret[0].(*core.IsolatedDeviceDesc)
	return ret0
}

// GetIsolatedDevice indicates an expected call of GetIsolatedDevice
func (mr *MockCandidatePropertyGetterMockRecorder) GetIsolatedDevice(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIsolatedDevice", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetIsolatedDevice), arg0)
}

// GetIsolatedDevices mocks base method
func (m *MockCandidatePropertyGetter) GetIsolatedDevices() []*core.IsolatedDeviceDesc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIsolatedDevices")
	ret0, _ := ret[0].([]*core.IsolatedDeviceDesc)
	return ret0
}

// GetIsolatedDevices indicates an expected call of GetIsolatedDevices
func (mr *MockCandidatePropertyGetterMockRecorder) GetIsolatedDevices() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIsolatedDevices", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetIsolatedDevices))
}

// GetPendingUsage mocks base method
func (m *MockCandidatePropertyGetter) GetPendingUsage() *models0.SPendingUsage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPendingUsage")
	ret0, _ := ret[0].(*models0.SPendingUsage)
	return ret0
}

// GetPendingUsage indicates an expected call of GetPendingUsage
func (mr *MockCandidatePropertyGetterMockRecorder) GetPendingUsage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPendingUsage", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetPendingUsage))
}

// GetQuotaKeys mocks base method
func (m *MockCandidatePropertyGetter) GetQuotaKeys(arg0 *api.SchedInfo) models.SComputeResourceKeys {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetQuotaKeys", arg0)
	ret0, _ := ret[0].(models.SComputeResourceKeys)
	return ret0
}

// GetQuotaKeys indicates an expected call of GetQuotaKeys
func (mr *MockCandidatePropertyGetterMockRecorder) GetQuotaKeys(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetQuotaKeys", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).GetQuotaKeys), arg0)
}

// Host mocks base method
func (m *MockCandidatePropertyGetter) Host() *models.SHost {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Host")
	ret0, _ := ret[0].(*models.SHost)
	return ret0
}

// Host indicates an expected call of Host
func (mr *MockCandidatePropertyGetterMockRecorder) Host() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Host", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Host))
}

// HostSchedtags mocks base method
func (m *MockCandidatePropertyGetter) HostSchedtags() []models.SSchedtag {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HostSchedtags")
	ret0, _ := ret[0].([]models.SSchedtag)
	return ret0
}

// HostSchedtags indicates an expected call of HostSchedtags
func (mr *MockCandidatePropertyGetterMockRecorder) HostSchedtags() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HostSchedtags", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).HostSchedtags))
}

// HostStatus mocks base method
func (m *MockCandidatePropertyGetter) HostStatus() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HostStatus")
	ret0, _ := ret[0].(string)
	return ret0
}

// HostStatus indicates an expected call of HostStatus
func (mr *MockCandidatePropertyGetterMockRecorder) HostStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HostStatus", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).HostStatus))
}

// HostType mocks base method
func (m *MockCandidatePropertyGetter) HostType() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HostType")
	ret0, _ := ret[0].(string)
	return ret0
}

// HostType indicates an expected call of HostType
func (mr *MockCandidatePropertyGetterMockRecorder) HostType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HostType", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).HostType))
}

// Id mocks base method
func (m *MockCandidatePropertyGetter) Id() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Id")
	ret0, _ := ret[0].(string)
	return ret0
}

// Id indicates an expected call of Id
func (mr *MockCandidatePropertyGetterMockRecorder) Id() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Id", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Id))
}

// InstanceGroups mocks base method
func (m *MockCandidatePropertyGetter) InstanceGroups() map[string]*api.CandidateGroup {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstanceGroups")
	ret0, _ := ret[0].(map[string]*api.CandidateGroup)
	return ret0
}

// InstanceGroups indicates an expected call of InstanceGroups
func (mr *MockCandidatePropertyGetterMockRecorder) InstanceGroups() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstanceGroups", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).InstanceGroups))
}

// IsEmpty mocks base method
func (m *MockCandidatePropertyGetter) IsEmpty() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsEmpty")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsEmpty indicates an expected call of IsEmpty
func (mr *MockCandidatePropertyGetterMockRecorder) IsEmpty() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsEmpty", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).IsEmpty))
}

// IsPublic mocks base method
func (m *MockCandidatePropertyGetter) IsPublic() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsPublic")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsPublic indicates an expected call of IsPublic
func (mr *MockCandidatePropertyGetterMockRecorder) IsPublic() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsPublic", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).IsPublic))
}

// KeywordPlural mocks base method
func (m *MockCandidatePropertyGetter) KeywordPlural() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "KeywordPlural")
	ret0, _ := ret[0].(string)
	return ret0
}

// KeywordPlural indicates an expected call of KeywordPlural
func (mr *MockCandidatePropertyGetterMockRecorder) KeywordPlural() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KeywordPlural", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).KeywordPlural))
}

// Name mocks base method
func (m *MockCandidatePropertyGetter) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockCandidatePropertyGetterMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Name))
}

// NetInterfaces mocks base method
func (m *MockCandidatePropertyGetter) NetInterfaces() map[string][]models.SNetInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NetInterfaces")
	ret0, _ := ret[0].(map[string][]models.SNetInterface)
	return ret0
}

// NetInterfaces indicates an expected call of NetInterfaces
func (mr *MockCandidatePropertyGetterMockRecorder) NetInterfaces() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NetInterfaces", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).NetInterfaces))
}

// Networks mocks base method
func (m *MockCandidatePropertyGetter) Networks() []*api.CandidateNetwork {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Networks")
	ret0, _ := ret[0].([]*api.CandidateNetwork)
	return ret0
}

// Networks indicates an expected call of Networks
func (mr *MockCandidatePropertyGetterMockRecorder) Networks() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Networks", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Networks))
}

// OvnCapable mocks base method
func (m *MockCandidatePropertyGetter) OvnCapable() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OvnCapable")
	ret0, _ := ret[0].(bool)
	return ret0
}

// OvnCapable indicates an expected call of OvnCapable
func (mr *MockCandidatePropertyGetterMockRecorder) OvnCapable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OvnCapable", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).OvnCapable))
}

// ProjectGuests mocks base method
func (m *MockCandidatePropertyGetter) ProjectGuests() map[string]int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProjectGuests")
	ret0, _ := ret[0].(map[string]int64)
	return ret0
}

// ProjectGuests indicates an expected call of ProjectGuests
func (mr *MockCandidatePropertyGetterMockRecorder) ProjectGuests() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProjectGuests", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).ProjectGuests))
}

// PublicScope mocks base method
func (m *MockCandidatePropertyGetter) PublicScope() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublicScope")
	ret0, _ := ret[0].(string)
	return ret0
}

// PublicScope indicates an expected call of PublicScope
func (mr *MockCandidatePropertyGetterMockRecorder) PublicScope() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublicScope", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).PublicScope))
}

// Region mocks base method
func (m *MockCandidatePropertyGetter) Region() *models.SCloudregion {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Region")
	ret0, _ := ret[0].(*models.SCloudregion)
	return ret0
}

// Region indicates an expected call of Region
func (mr *MockCandidatePropertyGetterMockRecorder) Region() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Region", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Region))
}

// ResourceType mocks base method
func (m *MockCandidatePropertyGetter) ResourceType() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResourceType")
	ret0, _ := ret[0].(string)
	return ret0
}

// ResourceType indicates an expected call of ResourceType
func (mr *MockCandidatePropertyGetterMockRecorder) ResourceType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResourceType", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).ResourceType))
}

// RunningCPUCount mocks base method
func (m *MockCandidatePropertyGetter) RunningCPUCount() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunningCPUCount")
	ret0, _ := ret[0].(int64)
	return ret0
}

// RunningCPUCount indicates an expected call of RunningCPUCount
func (mr *MockCandidatePropertyGetterMockRecorder) RunningCPUCount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunningCPUCount", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).RunningCPUCount))
}

// RunningMemorySize mocks base method
func (m *MockCandidatePropertyGetter) RunningMemorySize() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunningMemorySize")
	ret0, _ := ret[0].(int64)
	return ret0
}

// RunningMemorySize indicates an expected call of RunningMemorySize
func (mr *MockCandidatePropertyGetterMockRecorder) RunningMemorySize() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunningMemorySize", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).RunningMemorySize))
}

// SharedDomains mocks base method
func (m *MockCandidatePropertyGetter) SharedDomains() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SharedDomains")
	ret0, _ := ret[0].([]string)
	return ret0
}

// SharedDomains indicates an expected call of SharedDomains
func (mr *MockCandidatePropertyGetterMockRecorder) SharedDomains() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SharedDomains", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).SharedDomains))
}

// Sku mocks base method
func (m *MockCandidatePropertyGetter) Sku(arg0 string) *sku.ServerSku {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sku", arg0)
	ret0, _ := ret[0].(*sku.ServerSku)
	return ret0
}

// Sku indicates an expected call of Sku
func (mr *MockCandidatePropertyGetterMockRecorder) Sku(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sku", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Sku), arg0)
}

// Status mocks base method
func (m *MockCandidatePropertyGetter) Status() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status")
	ret0, _ := ret[0].(string)
	return ret0
}

// Status indicates an expected call of Status
func (mr *MockCandidatePropertyGetterMockRecorder) Status() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Status))
}

// StorageInfo mocks base method
func (m *MockCandidatePropertyGetter) StorageInfo() []*baremetal.BaremetalStorage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StorageInfo")
	ret0, _ := ret[0].([]*baremetal.BaremetalStorage)
	return ret0
}

// StorageInfo indicates an expected call of StorageInfo
func (mr *MockCandidatePropertyGetterMockRecorder) StorageInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StorageInfo", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).StorageInfo))
}

// Storages mocks base method
func (m *MockCandidatePropertyGetter) Storages() []*api.CandidateStorage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Storages")
	ret0, _ := ret[0].([]*api.CandidateStorage)
	return ret0
}

// Storages indicates an expected call of Storages
func (mr *MockCandidatePropertyGetterMockRecorder) Storages() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Storages", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Storages))
}

// TotalCPUCount mocks base method
func (m *MockCandidatePropertyGetter) TotalCPUCount(arg0 bool) int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TotalCPUCount", arg0)
	ret0, _ := ret[0].(int64)
	return ret0
}

// TotalCPUCount indicates an expected call of TotalCPUCount
func (mr *MockCandidatePropertyGetterMockRecorder) TotalCPUCount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TotalCPUCount", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).TotalCPUCount), arg0)
}

// TotalMemorySize mocks base method
func (m *MockCandidatePropertyGetter) TotalMemorySize(arg0 bool) int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TotalMemorySize", arg0)
	ret0, _ := ret[0].(int64)
	return ret0
}

// TotalMemorySize indicates an expected call of TotalMemorySize
func (mr *MockCandidatePropertyGetterMockRecorder) TotalMemorySize(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TotalMemorySize", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).TotalMemorySize), arg0)
}

// UnusedGpuDevices mocks base method
func (m *MockCandidatePropertyGetter) UnusedGpuDevices() []*core.IsolatedDeviceDesc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnusedGpuDevices")
	ret0, _ := ret[0].([]*core.IsolatedDeviceDesc)
	return ret0
}

// UnusedGpuDevices indicates an expected call of UnusedGpuDevices
func (mr *MockCandidatePropertyGetterMockRecorder) UnusedGpuDevices() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnusedGpuDevices", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).UnusedGpuDevices))
}

// UnusedIsolatedDevices mocks base method
func (m *MockCandidatePropertyGetter) UnusedIsolatedDevices() []*core.IsolatedDeviceDesc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnusedIsolatedDevices")
	ret0, _ := ret[0].([]*core.IsolatedDeviceDesc)
	return ret0
}

// UnusedIsolatedDevices indicates an expected call of UnusedIsolatedDevices
func (mr *MockCandidatePropertyGetterMockRecorder) UnusedIsolatedDevices() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnusedIsolatedDevices", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).UnusedIsolatedDevices))
}

// UnusedIsolatedDevicesByModel mocks base method
func (m *MockCandidatePropertyGetter) UnusedIsolatedDevicesByModel(arg0 string) []*core.IsolatedDeviceDesc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnusedIsolatedDevicesByModel", arg0)
	ret0, _ := ret[0].([]*core.IsolatedDeviceDesc)
	return ret0
}

// UnusedIsolatedDevicesByModel indicates an expected call of UnusedIsolatedDevicesByModel
func (mr *MockCandidatePropertyGetterMockRecorder) UnusedIsolatedDevicesByModel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnusedIsolatedDevicesByModel", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).UnusedIsolatedDevicesByModel), arg0)
}

// UnusedIsolatedDevicesByType mocks base method
func (m *MockCandidatePropertyGetter) UnusedIsolatedDevicesByType(arg0 string) []*core.IsolatedDeviceDesc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnusedIsolatedDevicesByType", arg0)
	ret0, _ := ret[0].([]*core.IsolatedDeviceDesc)
	return ret0
}

// UnusedIsolatedDevicesByType indicates an expected call of UnusedIsolatedDevicesByType
func (mr *MockCandidatePropertyGetterMockRecorder) UnusedIsolatedDevicesByType(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnusedIsolatedDevicesByType", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).UnusedIsolatedDevicesByType), arg0)
}

// UnusedIsolatedDevicesByVendorModel mocks base method
func (m *MockCandidatePropertyGetter) UnusedIsolatedDevicesByVendorModel(arg0 string) []*core.IsolatedDeviceDesc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnusedIsolatedDevicesByVendorModel", arg0)
	ret0, _ := ret[0].([]*core.IsolatedDeviceDesc)
	return ret0
}

// UnusedIsolatedDevicesByVendorModel indicates an expected call of UnusedIsolatedDevicesByVendorModel
func (mr *MockCandidatePropertyGetterMockRecorder) UnusedIsolatedDevicesByVendorModel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnusedIsolatedDevicesByVendorModel", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).UnusedIsolatedDevicesByVendorModel), arg0)
}

// Zone mocks base method
func (m *MockCandidatePropertyGetter) Zone() *models.SZone {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Zone")
	ret0, _ := ret[0].(*models.SZone)
	return ret0
}

// Zone indicates an expected call of Zone
func (mr *MockCandidatePropertyGetterMockRecorder) Zone() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Zone", reflect.TypeOf((*MockCandidatePropertyGetter)(nil).Zone))
}

// MockCandidater is a mock of Candidater interface
type MockCandidater struct {
	ctrl     *gomock.Controller
	recorder *MockCandidaterMockRecorder
}

// MockCandidaterMockRecorder is the mock recorder for MockCandidater
type MockCandidaterMockRecorder struct {
	mock *MockCandidater
}

// NewMockCandidater creates a new mock instance
func NewMockCandidater(ctrl *gomock.Controller) *MockCandidater {
	mock := &MockCandidater{ctrl: ctrl}
	mock.recorder = &MockCandidaterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCandidater) EXPECT() *MockCandidaterMockRecorder {
	return m.recorder
}

// GetGuestCount mocks base method
func (m *MockCandidater) GetGuestCount() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGuestCount")
	ret0, _ := ret[0].(int64)
	return ret0
}

// GetGuestCount indicates an expected call of GetGuestCount
func (mr *MockCandidaterMockRecorder) GetGuestCount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGuestCount", reflect.TypeOf((*MockCandidater)(nil).GetGuestCount))
}

// GetResourceType mocks base method
func (m *MockCandidater) GetResourceType() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResourceType")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetResourceType indicates an expected call of GetResourceType
func (mr *MockCandidaterMockRecorder) GetResourceType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResourceType", reflect.TypeOf((*MockCandidater)(nil).GetResourceType))
}

// GetSchedDesc mocks base method
func (m *MockCandidater) GetSchedDesc() *jsonutils.JSONDict {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSchedDesc")
	ret0, _ := ret[0].(*jsonutils.JSONDict)
	return ret0
}

// GetSchedDesc indicates an expected call of GetSchedDesc
func (mr *MockCandidaterMockRecorder) GetSchedDesc() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSchedDesc", reflect.TypeOf((*MockCandidater)(nil).GetSchedDesc))
}

// Getter mocks base method
func (m *MockCandidater) Getter() core.CandidatePropertyGetter {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Getter")
	ret0, _ := ret[0].(core.CandidatePropertyGetter)
	return ret0
}

// Getter indicates an expected call of Getter
func (mr *MockCandidaterMockRecorder) Getter() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Getter", reflect.TypeOf((*MockCandidater)(nil).Getter))
}

// IndexKey mocks base method
func (m *MockCandidater) IndexKey() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IndexKey")
	ret0, _ := ret[0].(string)
	return ret0
}

// IndexKey indicates an expected call of IndexKey
func (mr *MockCandidaterMockRecorder) IndexKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IndexKey", reflect.TypeOf((*MockCandidater)(nil).IndexKey))
}

// Type mocks base method
func (m *MockCandidater) Type() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Type")
	ret0, _ := ret[0].(int)
	return ret0
}

// Type indicates an expected call of Type
func (mr *MockCandidaterMockRecorder) Type() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Type", reflect.TypeOf((*MockCandidater)(nil).Type))
}

// MockScheduler is a mock of Scheduler interface
type MockScheduler struct {
	ctrl     *gomock.Controller
	recorder *MockSchedulerMockRecorder
}

// MockSchedulerMockRecorder is the mock recorder for MockScheduler
type MockSchedulerMockRecorder struct {
	mock *MockScheduler
}

// NewMockScheduler creates a new mock instance
func NewMockScheduler(ctrl *gomock.Controller) *MockScheduler {
	mock := &MockScheduler{ctrl: ctrl}
	mock.recorder = &MockSchedulerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockScheduler) EXPECT() *MockSchedulerMockRecorder {
	return m.recorder
}

// BeforePredicate mocks base method
func (m *MockScheduler) BeforePredicate() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BeforePredicate")
	ret0, _ := ret[0].(error)
	return ret0
}

// BeforePredicate indicates an expected call of BeforePredicate
func (mr *MockSchedulerMockRecorder) BeforePredicate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BeforePredicate", reflect.TypeOf((*MockScheduler)(nil).BeforePredicate))
}

// Predicates mocks base method
func (m *MockScheduler) Predicates() (map[string]core.FitPredicate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Predicates")
	ret0, _ := ret[0].(map[string]core.FitPredicate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Predicates indicates an expected call of Predicates
func (mr *MockSchedulerMockRecorder) Predicates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Predicates", reflect.TypeOf((*MockScheduler)(nil).Predicates))
}

// PriorityConfigs mocks base method
func (m *MockScheduler) PriorityConfigs() ([]core.PriorityConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PriorityConfigs")
	ret0, _ := ret[0].([]core.PriorityConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PriorityConfigs indicates an expected call of PriorityConfigs
func (mr *MockSchedulerMockRecorder) PriorityConfigs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PriorityConfigs", reflect.TypeOf((*MockScheduler)(nil).PriorityConfigs))
}

// MockINetworkNicCountGetter is a mock of INetworkNicCountGetter interface
type MockINetworkNicCountGetter struct {
	ctrl     *gomock.Controller
	recorder *MockINetworkNicCountGetterMockRecorder
}

// MockINetworkNicCountGetterMockRecorder is the mock recorder for MockINetworkNicCountGetter
type MockINetworkNicCountGetterMockRecorder struct {
	mock *MockINetworkNicCountGetter
}

// NewMockINetworkNicCountGetter creates a new mock instance
func NewMockINetworkNicCountGetter(ctrl *gomock.Controller) *MockINetworkNicCountGetter {
	mock := &MockINetworkNicCountGetter{ctrl: ctrl}
	mock.recorder = &MockINetworkNicCountGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockINetworkNicCountGetter) EXPECT() *MockINetworkNicCountGetterMockRecorder {
	return m.recorder
}

// GetTotalNicCount mocks base method
func (m *MockINetworkNicCountGetter) GetTotalNicCount(arg0 []string) (map[string]int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTotalNicCount", arg0)
	ret0, _ := ret[0].(map[string]int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTotalNicCount indicates an expected call of GetTotalNicCount
func (mr *MockINetworkNicCountGetterMockRecorder) GetTotalNicCount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTotalNicCount", reflect.TypeOf((*MockINetworkNicCountGetter)(nil).GetTotalNicCount), arg0)
}
