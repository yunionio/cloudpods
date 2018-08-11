package data_manager

import (
	candidatecache "yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	"yunion.io/x/pkg/utils"
)

type ResAlgorithm interface {
	Sum(values []value_t) value_t
	Subtract(sum value_t, value value_t) value_t
}

//////// //////// //////// //////// //////// ////////

type DefaultResAlgorithm struct {
}

func (al *DefaultResAlgorithm) Sum(values []value_t) value_t {
	var ret int64 = 0

	for _, value := range values {
		if value != nil {
			ret += value.(int64)
		}
	}

	return ret
}

func (al *DefaultResAlgorithm) Subtract(sum value_t, value value_t) value_t {
	if sum == nil {
		return nil
	}

	if value == nil {
		return sum
	}

	return sum.(int64) - value.(int64)
}

//////// //////// //////// //////// //////// ////////

type GroupResAlgorithmResult struct {
	Groups []*candidatecache.GroupCounts
}

func NewGroupResAlgorithmResult() *GroupResAlgorithmResult {
	return &GroupResAlgorithmResult{
		Groups: nil,
	}
}

func (r *GroupResAlgorithmResult) GuestCountOfGroup(groupId string) int64 {

	count := int64(0)

	for _, groupCounts := range r.Groups {
		if groupCount, ok := groupCounts.Data[groupId]; ok {
			count += groupCount.Count
		}
	}

	return count
}

func (r *GroupResAlgorithmResult) ExistsGroup(groupId string) bool {

	for _, groupCounts := range r.Groups {
		if groupCount, ok := groupCounts.Data[groupId]; ok {
			if groupCount.Count > 0 {
				return true
			}
		}
	}

	return false
}

type GroupResAlgorithm struct {
}

func (al *GroupResAlgorithm) Sum(values []value_t) value_t {
	r := NewGroupResAlgorithmResult()

	for _, value := range values {
		if value != nil {
			if v, ok := value.(*candidatecache.GroupCounts); ok {
				r.Groups = append(r.Groups, v)
			} else if v, ok := value.(*GroupResAlgorithmResult); ok {
				r.Groups = append(r.Groups, v.Groups...)
			}
		}
	}

	return r
}

func (al *GroupResAlgorithm) Subtract(sum value_t, value value_t) value_t {

	r := NewGroupResAlgorithmResult()

	if sum != nil {
		groups, _ := sum.(*candidatecache.GroupCounts)
		if groups != nil && len(groups.Data) > 0 {
			r.Groups = append(r.Groups, groups)
		}
	}

	if value != nil {
		grar, _ := value.(*GroupResAlgorithmResult)
		if grar != nil && len(grar.Groups) > 0 {
			r.Groups = append(r.Groups, grar.Groups...)
		}
	}

	return r
}

//////// //////// //////// //////// //////// ////////

type NetworksResAlgorithmResult struct {
	Networks map[string]int
}

type NetworksResAlgorithm struct {
}

func (a *NetworksResAlgorithm) Sum(values []value_t) value_t {
	countOfNetworks := 0
	for _, value := range values {
		if value != nil {
			if v, ok := value.(int); ok {
				countOfNetworks += v
			}
		}
	}

	return countOfNetworks
}

func (a *NetworksResAlgorithm) Subtract(sum value_t, value value_t) value_t {

	r := make(map[string]int)

	return r
}

type GroupGuestRelation struct {
	Data map[string]*candidatecache.GroupCount
}

//////// //////// //////// //////// //////// ////////

type IsolatedDeviceResAlgorithmResult struct {
	IDs map[string]int
}

func newIsolatedDeviceResAlgorithmResult() *IsolatedDeviceResAlgorithmResult {
	return &IsolatedDeviceResAlgorithmResult{
		IDs: make(map[string]int),
	}
}

func (r *IsolatedDeviceResAlgorithmResult) appendDevices(value value_t) {
	if devices, ok := value.([]*candidatecache.IsolatedDeviceDesc); ok {
		for _, dev := range devices {
			r.IDs[dev.ID] = 0
		}
	} else if r2, ok := value.(*IsolatedDeviceResAlgorithmResult); ok {
		for id := range r2.IDs {
			r.IDs[id] = 0
		}
	} else if ids, ok := value.([]string); ok {
		for _, id := range ids {
			r.IDs[id] = 0
		}
	}
}

func (r *IsolatedDeviceResAlgorithmResult) removeDevices(value value_t) {
	if devices, ok := value.([]*candidatecache.IsolatedDeviceDesc); ok {
		for _, dev := range devices {
			delete(r.IDs, dev.ID)
		}
	} else if r2, ok := value.(*IsolatedDeviceResAlgorithmResult); ok {
		for id := range r2.IDs {
			delete(r.IDs, id)
		}
	} else if ids, ok := value.([]string); ok {
		for _, id := range ids {
			delete(r.IDs, id)
		}
	}
}

type IsolatedDeviceResAlgorithm struct {
}

func (a *IsolatedDeviceResAlgorithm) Sum(values []value_t) value_t {
	r := newIsolatedDeviceResAlgorithmResult()

	for _, v := range values {
		r.appendDevices(v)
	}

	return r
}

func (al *IsolatedDeviceResAlgorithm) Subtract(sum value_t, value value_t) value_t {

	if sum == nil || value == nil {
		return sum
	}

	reserved := value.(*IsolatedDeviceResAlgorithmResult)
	var isolatedDevices []*candidatecache.IsolatedDeviceDesc
	for _, dev := range sum.([]*candidatecache.IsolatedDeviceDesc) {
		if _, ok := reserved.IDs[dev.ID]; !ok {
			isolatedDevices = append(isolatedDevices, dev)
		}
	}

	return isolatedDevices
}

//////// //////// //////// //////// //////// ////////

var (
	g_defaultResAlgorithm        *DefaultResAlgorithm        = &DefaultResAlgorithm{}
	g_groupResAlgorithm          *GroupResAlgorithm          = &GroupResAlgorithm{}
	g_networksResAlgorithm       *NetworksResAlgorithm       = &NetworksResAlgorithm{}
	g_isolatedDeviceResAlgorithm *IsolatedDeviceResAlgorithm = &IsolatedDeviceResAlgorithm{}
)

func GetResAlgorithm(res_name string) ResAlgorithm {
	switch res_name {
	case "Groups":
		return g_groupResAlgorithm
	case "IsolatedDevices":
		return g_isolatedDeviceResAlgorithm
	case "FreeCPUCount", "FreeMemSize", "FreeLocalStorageSize":
		return g_defaultResAlgorithm
	case "Ports":
		return g_defaultResAlgorithm
	default:
		if utils.HasPrefix(res_name, "FreeStorageSize:") {
			return g_defaultResAlgorithm
		}

		return nil
	}
}
