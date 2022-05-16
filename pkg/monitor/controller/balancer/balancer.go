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

package balancer

import (
	"sort"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	compute_options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

func init() {
	for _, drv := range []IMetricDriver{
		newMemAvailable(),
		newCPUUsageActive(),
	} {
		GetMetricDrivers().register(drv)
	}
}

var (
	drivers *MetricDrivers
)

type MetricDrivers struct {
	*sync.Map
}

func NewMetricDrivers() *MetricDrivers {
	return &MetricDrivers{
		Map: new(sync.Map),
	}
}

func (d *MetricDrivers) register(id IMetricDriver) *MetricDrivers {
	d.Store(id.GetType(), id)
	return d
}

func (d *MetricDrivers) Get(mT monitor.MigrationAlertMetricType) (IMetricDriver, error) {
	drv, ok := d.Load(mT)
	if !ok {
		return nil, errors.Errorf("Not found driver by %q", mT)
	}
	return drv.(IMetricDriver), nil
}

func GetMetricDrivers() *MetricDrivers {
	if drivers == nil {
		drivers = NewMetricDrivers()
	}
	return drivers
}

type IMetricDriver interface {
	GetType() monitor.MigrationAlertMetricType
	GetTsdbQuery() *TsdbQuery
	GetCandidate(gst jsonutils.JSONObject, host IHost, ds *tsdb.DataSource) (ICandidate, error)
	SetHostCurrent(h IHost, values map[string]float64) error
	GetTarget(host jsonutils.JSONObject) (ITarget, error)
	GetCondition(*monitor.EvalMatch) ICondition
}

type ICondition interface {
	GetThreshold() float64
	IsFitTarget(t ITarget, c ICandidate) error
}

type Rules struct {
	Condtion ICondition
	Source   *SourceRule
	Target   *TargetRule
}

func NewRules(ctx *alerting.EvalContext, m *monitor.EvalMatch, drv IMetricDriver) (*Rules, error) {
	hostId, ok := m.Tags["host_id"]
	if !ok {
		return nil, errors.Errorf("Not found host_id in tags: %#v", m.Tags)
	}
	ok, hObjs := models.MonitorResourceManager.GetResourceObjByResType(monitor.METRIC_RES_TYPE_HOST)
	if !ok {
		return nil, errors.Errorf("GetResourceObjByResType host returns false")
	}
	var hObj jsonutils.JSONObject = nil
	for _, obj := range hObjs {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "get host obj id: %s", obj)
		}
		if id == hostId {
			hObj = obj
			break
		}
	}
	if hObj == nil {
		return nil, errors.Errorf("Not found source host object by id: %q, %q", hostId, hObj)
	}
	host, err := newMemHost(hObj)
	if err != nil {
		return nil, errors.Wrap(err, "new host")
	}
	allHosts := []IResource{host}

	otherHosts := make([]ITarget, 0)
	for _, obj := range hObjs {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "get host obj id: %s", obj)
		}
		if id == hostId {
			continue
		} else {
			hostType, _ := obj.GetString("host_type")
			if hostType != computeapi.HOST_TYPE_HYPERVISOR {
				// only treat on premise host as target
				continue
			}
			th, err := drv.GetTarget(obj)
			if err != nil {
				return nil, errors.Wrapf(err, "drv.GetTarget %s", obj)
			}
			otherHosts = append(otherHosts, th)
			allHosts = append(allHosts, th)
		}
	}
	ds := &tsdb.DataSource{
		Type: "influxdb",
		Name: "default",
		Url:  "https://10.127.100.2:30086",
	}
	cds, err := findGuestsOfHost(drv, host, ds)
	if err != nil {
		return nil, errors.Wrapf(err, "findGuestsOfHost %s", host.GetName())
	}
	metrics, err := InfluxdbQuery(ds, "host_id", allHosts, drv.GetTsdbQuery())
	if err != nil {
		return nil, errors.Wrapf(err, "InfluxdbQuery all hosts metrics")
	}
	for _, host := range allHosts {
		m := metrics.Get(host.GetId())
		if m == nil {
			return nil, errors.Errorf("Influxdb metrics of %s(%s) not found", host.GetName(), host.GetId())
		}
		if err := drv.SetHostCurrent(host.(IHost), m.Values); err != nil {
			return nil, errors.Wrapf(err, "SetHostCurrent %q", host.GetName())
		}
	}
	rs := &Rules{
		Condtion: drv.GetCondition(m),
	}
	rs.Source = NewSourceRule(host, cds)
	rs.Target = NewTargetRule(otherHosts)
	return rs, nil
}

func findGuestsOfHost(drv IMetricDriver, host IHost, ds *tsdb.DataSource) ([]ICandidate, error) {
	ok, objs := models.MonitorResourceManager.GetResourceObjByResType(monitor.METRIC_RES_TYPE_GUEST)
	if !ok {
		return nil, errors.Errorf("GetResourceObjByResType by guest return false")
	}
	ret := make([]ICandidate, 0)
	for _, obj := range objs {
		gHostId, err := obj.GetString("host_id")
		if err != nil {
			return nil, errors.Wrapf(err, "get host_id from cache guest %s", obj)
		}
		if gHostId == host.GetId() {
			status, err := obj.GetString("status")
			if err != nil {
				return nil, errors.Wrapf(err, "get status of guest: %s", obj)
			}
			// filter running guest
			if status != computeapi.VM_RUNNING {
				continue
			}
			c, err := drv.GetCandidate(obj, host, ds)
			if err != nil {
				return nil, errors.Wrapf(err, "drv.GetCandidate of guest %s", obj)
			}
			ret = append(ret, c)
		}
	}
	return ret, nil
}

// SourceRule 定义触发了报警的宿主机和上面可以迁移的虚拟机
type SourceRule struct {
	Host       IHost
	Candidates []ICandidate
}

func NewSourceRule(host IHost, cds []ICandidate) *SourceRule {
	return &SourceRule{
		Host:       host,
		Candidates: cds,
	}
}

type IHost interface {
	IResource
	GetHostResource() *HostResource
	GetCurrent() float64
	SetCurrent(float64) IHost
	Compare(oh IHost) bool
}

// TargetRule 定义可以选择迁移的宿主机
type TargetRule struct {
	Items []ITarget
}

func NewTargetRule(hosts []ITarget) *TargetRule {
	return &TargetRule{
		Items: hosts,
	}
}

type ItemType string

const (
	ItemTypeHost  = "host"
	ItemTypeGuest = "guest"
)

type ITarget interface {
	IHost
	Selected(c ICandidate) ITarget
}

type iTargets []ITarget

func (i iTargets) Len() int {
	return len(i)
}

func (ts iTargets) Less(i, j int) bool {
	a, b := ts[i], ts[j]
	return a.Compare(b)
}

func (ts iTargets) Swap(i, j int) {
	ts[i], ts[j] = ts[j], ts[i]
}

func DoBalance(s *mcclient.ClientSession, rules *Rules) error {
	rst, err := findResult(rules)
	if err != nil {
		return errors.Wrapf(err, "find result to migrate")
	}

	if err := doMigrate(s, rst); err != nil {
		return errors.Wrapf(err, "do migrate for result %#v", rst)
	}

	return nil
}

type result struct {
	pairs []*resultPair
}

type resultPair struct {
	source ICandidate
	target ITarget
}

func findResult(rules *Rules) (*result, error) {
	// 找到 rules.Source 里面可以迁移的虚拟机
	// 只要迁移的虚拟机到其他宿主机后当前指标小于阈值
	log.Errorf("==findResult rules: %#v", rules)
	guests, err := findCandidates(rules.Source, rules.Condtion)
	if err != nil {
		return nil, errors.Wrap(err, "find source candidates to migrate")
	}

	// 将找到的虚拟机分配到对应的宿主机，形成 1-1 配对
	return pairMigratResult(guests, rules.Target, rules.Condtion)
}

type IResource interface {
	GetId() string
	GetName() string
}

type ICandidate interface {
	IResource
	GetScore() float64
}

func getCScores(css []ICandidate) []float64 {
	ret := make([]float64, len(css))
	for i := range css {
		ret[i] = css[i].GetScore()
	}
	return ret
}

func findFitCandidates(css []ICandidate, delta float64) ([]ICandidate, error) {
	if len(css) == 0 {
		return nil, errors.Errorf("Not found fit input for delta %f", delta)
	}
	first := css[0]
	rest := css[1:]
	if first.GetScore() >= delta {
		return []ICandidate{first}, nil
	}
	rRests, err := findFitCandidates(rest, delta-first.GetScore())
	if err != nil {
		return nil, errors.Wrapf(err, "Found in rest %v", rest)
	}
	ret := []ICandidate{first}
	ret = append(ret, rRests...)
	return ret, nil
}

func findCandidates(src *SourceRule, cond ICondition) ([]ICandidate, error) {
	threshold := cond.GetThreshold()
	hostCur := src.Host.GetCurrent()
	delta := threshold - hostCur
	return findFitCandidates(src.Candidates, delta)
}

func findFitTarget(c ICandidate, targets iTargets, cond ICondition) (ITarget, error) {
	// sort targets
	// find target - (metric of guest) > threshold
	sort.Sort(targets)
	var errs []error
	for i := range targets {
		target := targets[i]
		if target.GetCurrent()-c.GetScore() > cond.GetThreshold() {
			return target, nil
		} else {
			errs = append(errs, errors.Errorf(
				"Host:%s:current(%f) - candiate:%s:score(%f) <= %f(threshold)",
				target.GetName(), target.GetCurrent(),
				c.GetName(), c.GetScore(),
				cond.GetThreshold()))
		}
	}
	return nil, errors.NewAggregate(errs)
}

func pairMigratResult(gsts []ICandidate, target *TargetRule, cond ICondition) (*result, error) {
	pairs := make([]*resultPair, 0)
	hosts := target.Items
	for _, gst := range gsts {
		host, err := findFitTarget(gst, hosts, cond)
		if err != nil {
			return nil, errors.Wrapf(err, "not found target for guest %#v", gst)
		}
		host.Selected(gst)
		pairs = append(pairs, &resultPair{
			source: gst,
			target: host,
		})
	}
	if len(gsts) != len(pairs) {
		return nil, errors.Errorf("Paired: %d candidates != %d hosts", len(gsts), len(pairs))
	}
	return &result{
		pairs: pairs,
	}, nil
}

func doMigrate(s *mcclient.ClientSession, rst *result) error {
	// migrate must be executed on by one
	for _, pair := range rst.pairs {
		if obj, err := doMigrateByPair(s, pair); err != nil {
			return errors.Wrapf(err, "doMigrateByPair %#v", pair)
		} else {
			log.Infof("==Migrate server %s to %s", obj, pair.target.GetId())
		}
	}
	return nil
}

func doMigrateByPair(s *mcclient.ClientSession, pair *resultPair) (jsonutils.JSONObject, error) {
	gst := pair.source
	trueObj := true
	input := &compute_options.ServerLiveMigrateOptions{
		ID:              gst.GetId(),
		PreferHost:      pair.target.GetId(),
		SkipCpuCheck:    &trueObj,
		SkipKernelCheck: &trueObj,
	}
	params, err := input.Params()
	if err != nil {
		return nil, errors.Wrapf(err, "live migrate input %#v", input)
	}
	obj, err := compute.Servers.PerformAction(s, input.GetId(), "live-migrate", params)
	if err != nil {
		return nil, errors.Wrapf(err, "live migrate with params: %s", params)
	}
	return obj, nil
}

type resource struct {
	id   string
	name string
}

func newResource(obj jsonutils.JSONObject) (IResource, error) {
	id, err := obj.GetString("id")
	if err != nil {
		return nil, errors.Wrap(err, "get id")
	}
	name, err := obj.GetString("name")
	if err != nil {
		return nil, errors.Wrap(err, "get name")
	}
	return &resource{
		id:   id,
		name: name,
	}, nil
}

func (r *resource) GetId() string {
	return r.id
}

func (r *resource) GetName() string {
	return r.name
}

type guestResource struct {
	IResource
	guest jsonutils.JSONObject
}

func newGuestResource(gst jsonutils.JSONObject) (*guestResource, error) {
	res, err := newResource(gst)
	if err != nil {
		return nil, errors.Wrap(err, "newResource")
	}
	return &guestResource{
		IResource: res,
		guest:     gst,
	}, nil
}

type HostResource struct {
	IResource
	host         jsonutils.JSONObject
	totalMemSize float64
	cpuCount     int64
}

func newHostResource(host jsonutils.JSONObject) (*HostResource, error) {
	res, err := newResource(host)
	if err != nil {
		return nil, errors.Wrap(err, "newResource")
	}
	memSize, err := host.Int("mem_size")
	if err != nil {
		return nil, errors.Wrap(err, "get mem_size")
	}
	cpuCount, err := host.Int("cpu_count")
	if err != nil {
		return nil, errors.Wrap(err, "get cpu_count")
	}
	return &HostResource{
		IResource:    res,
		host:         host,
		totalMemSize: float64(memSize * 1024 * 1024),
		cpuCount:     cpuCount,
	}, nil
}

func (h *HostResource) GetHostResource() *HostResource {
	return h
}
