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
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

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
	GetCondition(s *monitor.AlertSetting) (ICondition, error)
}

type ICondition interface {
	GetThreshold() float64
	// GetSourceThresholdDelta must > 0
	GetSourceThresholdDelta(threshold float64, srcHost IHost) float64
	IsFitTarget(settings *monitor.MigrationAlertSettings, t ITarget, c ICandidate) error
}

type Rules struct {
	Alert          *models.SMigrationAlert
	Condtion       ICondition
	Source         *SourceRule
	Target         *TargetRule
	ResultMustPair bool
}

func (r *Rules) GetAlert() *models.SMigrationAlert {
	return r.Alert
}

func NewRules(_ *alerting.EvalContext, m *monitor.EvalMatch, alert *models.SMigrationAlert, drv IMetricDriver, resultMustPair bool) (*Rules, error) {
	hostId, ok := m.Tags["host_id"]
	if !ok {
		return nil, errors.Errorf("Not found host_id in tags: %#v", m.Tags)
	}
	ok, hObjs := models.MonitorResourceManager.GetResourceObjByResType(monitor.METRIC_RES_TYPE_HOST)
	if !ok {
		return nil, errors.Errorf("GetResourceObjByResType host returns false")
	}
	var srcHostObj jsonutils.JSONObject = nil
	for _, obj := range hObjs {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "get host obj id: %s", obj)
		}
		if id == hostId {
			srcHostObj = obj
			break
		}
	}
	if srcHostObj == nil {
		return nil, errors.Errorf("Not found source host object by id: %q, %q", hostId, srcHostObj)
	}
	srcHost, err := drv.GetTarget(srcHostObj)
	if err != nil {
		return nil, errors.Wrap(err, "new host")
	}

	allHosts := []IResource{srcHost}
	msettings, _ := alert.GetMigrationSettings()
	targetHosts, err := filterTargetHosts(drv, srcHost, hObjs, msettings)
	if err != nil {
		return nil, errors.Wrap(err, "filterTargetHosts")
	}
	for _, oh := range targetHosts {
		allHosts = append(allHosts, oh)
	}

	dsObj, err := models.DataSourceManager.GetDefaultSource()
	if err != nil {
		return nil, errors.Wrapf(err, "Get default DataSource")
	}
	ds := dsObj.ToTSDBDataSource("")
	// find guests to filtered by source setting of source host alerted
	cds, err := findGuestsOfHost(drv, srcHost, ds, msettings)
	if err != nil {
		return nil, errors.Wrapf(err, "findGuestsOfHost %s", srcHost.GetName())
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
	settings, err := alert.GetSettings()
	if err != nil {
		return nil, errors.Wrapf(err, "Get alert settings")
	}
	cond, err := drv.GetCondition(settings)
	if err != nil {
		return nil, errors.Wrapf(err, "Get Condtion")
	}
	rs := &Rules{
		Alert:          alert,
		Condtion:       cond,
		ResultMustPair: resultMustPair,
	}
	rs.Source = NewSourceRule(srcHost, cds)
	rs.Target = NewTargetRule(targetHosts)
	return rs, nil
}

func filterTargetHosts(drv IMetricDriver, srcHost IHost, allHost []jsonutils.JSONObject, ms *monitor.MigrationAlertSettings) ([]ITarget, error) {
	specifyTargetHostIds := []string{}
	specifySrcHostIds := []string{}
	if ms != nil && ms.Target != nil {
		specifyTargetHostIds = ms.Target.HostIds
	}
	if ms != nil && ms.Source != nil {
		specifySrcHostIds = ms.Source.HostIds
	}
	srcHostId := srcHost.GetId()
	srcHostObj := srcHost.GetObject()
	srcHostType, err := srcHostObj.GetString("host_type")
	if err != nil {
		return nil, errors.Wrap(err, "get source host_type")
	}
	srcArch, err := srcHostObj.GetString("cpu_architecture")
	if err != nil {
		return nil, errors.Wrapf(err, "get source cpu_architecture")
	}
	targets := make([]ITarget, 0)
	for _, obj := range allHost {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "get host obj id: %s", obj)
		}
		if id == srcHostId {
			continue
		}
		if len(specifySrcHostIds) != 0 {
			if utils.IsInStringArray(id, specifySrcHostIds) {
				// filter target host if it in source specified hosts
				continue
			}
		}
		if len(specifyTargetHostIds) != 0 {
			// filter target host if it not in target specified hosts
			if !utils.IsInStringArray(id, specifyTargetHostIds) {
				continue
			}
		}
		hostType, _ := obj.GetString("host_type")
		if hostType != srcHostType {
			continue
		}
		arch, _ := obj.GetString("cpu_architecture")
		if arch != srcArch {
			continue
		}

		enabled, _ := obj.Bool("enabled")
		if !enabled {
			continue
		}

		hostStatus, _ := obj.GetString("host_status")
		if hostStatus != "online" {
			continue
		}

		th, err := drv.GetTarget(obj)
		if err != nil {
			return nil, errors.Wrapf(err, "drv.GetTarget %s", obj)
		}
		targets = append(targets, th)
	}
	return targets, nil
}

func findGuestsOfHost(drv IMetricDriver, host IHost, ds *tsdb.DataSource, ms *monitor.MigrationAlertSettings) ([]ICandidate, error) {
	ok, objs := models.MonitorResourceManager.GetResourceObjByResType(monitor.METRIC_RES_TYPE_GUEST)
	if !ok {
		return nil, errors.Errorf("GetResourceObjByResType by guest return false")
	}

	specifyGuestIds := []string{}
	if ms != nil && ms.Source != nil {
		specifyGuestIds = ms.Source.GuestIds
	}

	ret := make([]ICandidate, 0)
	found := false
	errs := []error{}
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
			name, _ := obj.GetString("name")
			// filter running guest
			if status != computeapi.VM_RUNNING {
				log.Debugf("ignore guest %s cause status is %s", name, status)
				continue
			}
			gId, _ := obj.GetString("id")
			if len(specifyGuestIds) != 0 {
				if !utils.IsInStringArray(gId, specifyGuestIds) {
					log.Debugf("ignore guest %s(%s) cause not in specified ids %v", name, gId, specifyGuestIds)
					continue
				}
			}
			c, err := drv.GetCandidate(obj, host, ds)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "drv.GetCandidate of guest %s", obj))
				continue

			}
			if c.GetScore() == 0 {
				log.Debugf("ignore guest %s cause %s score is 0", c.GetName(), drv.GetType())
				continue
			}
			ret = append(ret, c)
			found = true
		}
	}
	if !found {
		return nil, errors.NewAggregate(errs)
	}
	if len(errs) != 0 {
		log.Warningf("not all guests found: %s", errors.NewAggregate(errs))
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

func RecoverInProcessAlerts(ctx context.Context, s *mcclient.ClientSession) error {
	alerts, err := models.GetMigrationAlertManager().GetInMigrationAlerts()
	if err != nil {
		return errors.Wrap(err, "GetInMigrationAlerts")
	}
	recorder := NewRecorder()
	errs := make([]error, 0)
	for _, alert := range alerts {
		notes, err := alert.GetMigrateNotes()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "GetMigrateNotes for %s", alert.GetId()))
			continue
		}
		for _, note := range notes {
			log.Infof("Start recover alert %s(%s) note %s", alert.GetName(), alert.GetId(), jsonutils.Marshal(note))
			notePrt := &note
			recorder.StartWatchMigratingProcess(ctx, s, alert, notePrt)
		}
	}
	return errors.NewAggregate(errs)
}

func DoBalance(ctx context.Context, s *mcclient.ClientSession, rules *Rules, recorder IRecorder) error {
	// check whether having migration in process
	alerts, err := models.GetMigrationAlertManager().GetInMigrationAlerts()
	if err != nil {
		return errors.Wrap(err, "GetInMigrationAlerts")
	}
	if len(alerts) != 0 {
		ids := make([]string, len(alerts))
		for i := range alerts {
			alert := alerts[i]
			ids[i] = fmt.Sprintf("%s(%s)", alert.GetName(), alert.GetId())
		}
		return errors.Errorf("Others migration alerts in process: %v", ids)
	}
	rst, err := findResult(rules)
	if err != nil {
		err = errors.Wrapf(err, "find result to migrate")
		recorder.RecordError(s.GetToken(), rules.GetAlert(), err, EventActionFindResultFail)
		return err
	}

	if err := doMigrate(ctx, s, rules, rst, recorder); err != nil {
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
	// guests, err := findCandidates(rules.Source, rules.Condtion)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "find source candidates to migrate")
	// }

	// TODO
	// 将找到的 guests 进行配对，调用 scheduler-forecast 接口判断能否迁移到宿主机
	// 如果不能迁就提出这些 guests，重新 findCandidates

	// 将找到的虚拟机分配到对应的宿主机，形成 1-1 配对
	settings, _ := rules.GetAlert().GetMigrationSettings()
	return pairMigratResult(settings, rules.Source, rules.Target, rules.Condtion, rules.ResultMustPair)
}

type IResource interface {
	GetId() string
	GetName() string
	GetObject() jsonutils.JSONObject
}

type ICandidate interface {
	IResource
	GetHostName() string
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
		return nil, errors.Errorf("Not found fit guest candidate for delta %f", delta)
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

type candidatesThresholdSort struct {
	cds       []ICandidate
	threshold float64
}

func newCandidatesThresholdSort(cds []ICandidate, th float64) *candidatesThresholdSort {
	return &candidatesThresholdSort{
		cds:       cds,
		threshold: th,
	}
}

func (s *candidatesThresholdSort) Len() int {
	return len(s.cds)
}

func (s *candidatesThresholdSort) Swap(i, j int) {
	s.cds[i], s.cds[j] = s.cds[j], s.cds[i]
}

func (s *candidatesThresholdSort) Less(i, j int) bool {
	d1 := math.Abs(s.cds[i].GetScore() - s.threshold)
	d2 := math.Abs(s.cds[j].GetScore() - s.threshold)
	return d1 < d2
}

func (s *candidatesThresholdSort) CandidatesString() string {
	str := fmt.Sprintf("delta: %f\n", s.threshold)
	for _, c := range s.cds {
		str += fmt.Sprintf("%s: %f\n", c.GetName(), c.GetScore())
	}
	return str
}

func (s *candidatesThresholdSort) Debug(prefix string) {
	log.Infof("%s:\n%s", prefix, s.CandidatesString())
}

func sortCandidatesByThreshold(cds []ICandidate, th float64) []ICandidate {
	ss := newCandidatesThresholdSort(cds, th)
	// ss.Debug("Pre sort")
	sort.Sort(ss)
	// ss.Debug("After sort")
	return ss.cds
}

func findCandidates(src *SourceRule, cond ICondition) ([]ICandidate, error) {
	threshold := cond.GetThreshold()
	delta := cond.GetSourceThresholdDelta(threshold, src.Host)
	cds := sortCandidatesByThreshold(src.Candidates, threshold)
	return findFitCandidates(cds, delta)
}

func findFitTarget(settings *monitor.MigrationAlertSettings, c ICandidate, tr *TargetRule, targets iTargets, cond ICondition) (ITarget, error) {
	// sort targets
	sort.Sort(targets)
	var errs []error
	for i := range targets {
		target := targets[i]
		if err := cond.IsFitTarget(settings, target, c); err == nil {
			return target, nil
		} else {
			errs = append(errs, err)
		}
	}
	return nil, errors.NewAggregate(errs)
}

func pairMigratResult(
	settings *monitor.MigrationAlertSettings,
	src *SourceRule, target *TargetRule, cond ICondition, mustPair bool) (*result, error) {
	// all guests of source host to migrate
	gsts := src.Candidates

	pairs := make([]*resultPair, 0)
	hosts := target.Items
	errs := []error{}
	for _, gst := range gsts {
		host, err := findFitTarget(settings, gst, target, hosts, cond)
		if err != nil {
			err = errors.Wrapf(err, "not found target for guest %s on %s", gst.GetName(), gst.GetHostName())
			if mustPair {
				return nil, err
			} else {
				errs = append(errs, err)
				continue
			}
		}
		host.Selected(gst)
		pairs = append(pairs, &resultPair{
			source: gst,
			target: host,
		})
	}
	if len(gsts) != len(pairs) {
		if mustPair {
			return nil, errors.Errorf("%v: Paired: %d candidates != %d hosts", errors.NewAggregate(errs), len(gsts), len(pairs))
		}
	}
	if len(pairs) == 0 {
		return nil, errors.Errorf("%v: Not found any pairs, mustPair is %v", errors.NewAggregate(errs), mustPair)
	}
	if !mustPair && len(errs) != 0 {
		log.Warningf("some guest not paired: %v", errors.NewAggregate(errs))
	}
	return &result{
		pairs: pairs,
	}, nil
}

func doMigrate(ctx context.Context, s *mcclient.ClientSession, rules *Rules, rst *result, recorder IRecorder) error {
	// migrate must be executed on by one
	alert := rules.GetAlert()
	for _, pair := range rst.pairs {
		note, err := NewMigrateNote(pair, nil)
		if err != nil {
			return errors.Wrap(err, "NewMigrateNotes")
		}
		if obj, err := doMigrateByPair(s, pair); err != nil {
			err = errors.Wrapf(err, "doMigrateByPair %s to %s", pair.source.GetName(), pair.target.GetName())
			if rErr := recorder.RecordMigrateError(s.GetToken(), alert, note, err); rErr != nil {
				log.Errorf("RecordMigrate %s to %s error: %v", obj, pair.target.GetId(), rErr)
			}
			return err
		} else {
			if err := recorder.RecordMigrate(ctx, s, alert, note); err != nil {
				log.Errorf("RecordMigrate %s to %s error: %v", obj, pair.target.GetId(), err)
			}
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
	obj  jsonutils.JSONObject
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
		obj:  obj,
	}, nil
}

func (r *resource) GetId() string {
	return r.id
}

func (r *resource) GetName() string {
	return r.name
}

func (r *resource) GetObject() jsonutils.JSONObject {
	return r.obj
}

type guestResource struct {
	IResource
	hostName string
	guest    jsonutils.JSONObject
}

func newGuestResource(gst jsonutils.JSONObject, hostName string) (*guestResource, error) {
	res, err := newResource(gst)
	if err != nil {
		return nil, errors.Wrap(err, "newResource")
	}
	return &guestResource{
		IResource: res,
		hostName:  hostName,
		guest:     gst,
	}, nil
}

func (s *guestResource) GetHostName() string {
	return s.hostName
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
