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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

func setHostCurrent(host IHost, vals map[string]float64, key string) error {
	val, ok := vals[key]
	if !ok {
		return errors.Errorf("not found %q in vals %#v", key, vals)
	}
	host.SetCurrent(val)
	return nil
}

type cpuUsageActive struct{}

func newCPUUsageActive() IMetricDriver {
	return &cpuUsageActive{}
}

func (c *cpuUsageActive) GetType() monitor.MigrationAlertMetricType {
	return monitor.MigrationAlertMetricTypeCPUUsageActive
}

func (c *cpuUsageActive) GetTsdbQuery() *TsdbQuery {
	return &TsdbQuery{
		Database:    monitor.METRIC_DATABASE_TELE,
		Measurement: "cpu",
		Fields:      []string{"usage_active"},
	}
}

func (c *cpuUsageActive) GetCandidate(obj jsonutils.JSONObject, host IHost, ds *tsdb.DataSource) (ICandidate, error) {
	return newCPUCandidate(obj, host.GetHostResource(), ds)
}

func (c *cpuUsageActive) SetHostCurrent(host IHost, vals map[string]float64) error {
	return setHostCurrent(host, vals, "usage_active")
}

func (c *cpuUsageActive) GetTarget(host jsonutils.JSONObject) (ITarget, error) {
	return newTargetCPUHost(host)
}

func (ma *cpuUsageActive) GetCondition(s *monitor.AlertSetting) (ICondition, error) {
	t, err := GetAlertSettingThreshold(s)
	if err != nil {
		return nil, errors.Wrap(err, "GetAlertSettingThreshold")
	}
	return newCPUCond(t), nil
}

// cpuCondition implements ICondition
type cpuCondition struct {
	value float64
}

func newCPUCond(val float64) ICondition {
	return &cpuCondition{
		value: val,
	}
}

func (c *cpuCondition) GetThreshold() float64 {
	return c.value
}

func (c *cpuCondition) GetSourceThresholdDelta(threshold float64, host IHost) float64 {
	// cpu.usage_active
	return host.GetCurrent() - threshold
}

func (m *cpuCondition) IsFitTarget(settings *monitor.MigrationAlertSettings, t ITarget, c ICandidate) error {
	src := settings.Source
	srcHostIds := []string{}
	if src != nil {
		srcHostIds = src.HostIds
	}
	tCPUCnt := t.(*targetCPUHost).GetCPUCount()
	tScore := t.GetCurrent() + c.(*cpuCandidate).getTargetScore(tCPUCnt)
	ltThreshold := tScore < m.GetThreshold()
	if ltThreshold {
		return nil
	}

	MAX_THRESHOLD := 95.0
	// only when srcHostIds isn't empty
	if len(srcHostIds) != 0 {
		if !ltThreshold && !sets.NewString(srcHostIds...).Has(t.GetId()) && tScore < MAX_THRESHOLD {
			// if target host is not in source specified hosts and calculated score is less than MAX_THRESHOLD
			log.Infof("let host:%s:current(%f) + guest:%s:score(%f) < MAX_THRESHOLD(%f) to fit target, because it's not in source specified hosts", t.GetName(), t.GetCurrent(), c.GetName(), c.GetScore(), MAX_THRESHOLD)
			return nil
		}
	}
	return errors.Errorf("host:%s:current(%f) + guest:%s:score(%f) >= threshold(%f)", t.GetName(), t.GetCurrent(), c.GetName(), c.GetScore(), m.GetThreshold())
}

// cpuCandidate implements ICandidate
type cpuCandidate struct {
	*guestResource
	usageActive   float64
	guestCPUCount int
	hostCPUCount  int
}

func newCPUCandidate(gst jsonutils.JSONObject, host *HostResource, ds *tsdb.DataSource) (ICandidate, error) {
	res, err := newGuestResource(gst, host.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "newGuestResource")
	}

	gstCPUCount, err := res.guest.Int("vcpu_count")
	if err != nil {
		return nil, errors.Wrap(err, "get vcpu_count")
	}

	// fetch metric from influxdb
	metrics, err := InfluxdbQuery(ds, "vm_id", []IResource{res}, &TsdbQuery{
		Database:    monitor.METRIC_DATABASE_TELE,
		Measurement: "vm_cpu",
		Fields:      []string{"usage_active"},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "InfluxdbQuery guest %q(%q)", res.GetName(), res.GetId())
	}

	metric := metrics.Get(res.GetId())
	if metric == nil {
		return nil, errors.Errorf("not found resource %q metric from %#v", res.GetId(), metrics.indexes)
	}
	usage := metric.Values["usage_active"]
	return &cpuCandidate{
		guestResource: res,
		usageActive:   usage,
		guestCPUCount: int(gstCPUCount),
		hostCPUCount:  int(host.cpuCount),
	}, nil
}

func (c cpuCandidate) GetScore() float64 {
	return c.getTargetScore(c.hostCPUCount)
}

func (c cpuCandidate) getTargetScore(tCPUCnt int) float64 {
	score := c.usageActive * (float64(c.guestCPUCount) / float64(tCPUCnt))
	return score
}

type cpuHost struct {
	*HostResource
	usageActive float64
}

func newCPUHost(obj jsonutils.JSONObject) (IHost, error) {
	host, err := newHostResource(obj)
	if err != nil {
		return nil, errors.Wrap(err, "newHostResource")
	}
	return &cpuHost{
		HostResource: host,
	}, nil
}

func (h *cpuHost) GetCurrent() float64 {
	return h.usageActive
}

func (h *cpuHost) SetCurrent(val float64) IHost {
	h.usageActive = val
	return h
}

func (h *cpuHost) Compare(oh IHost) bool {
	return h.GetCurrent() < oh.GetCurrent()
}

type targetCPUHost struct {
	IHost
}

func newTargetCPUHost(obj jsonutils.JSONObject) (ITarget, error) {
	host, err := newCPUHost(obj)
	if err != nil {
		return nil, errors.Wrap(err, "newCPUHost")
	}
	ts := &targetCPUHost{
		IHost: host,
	}
	return ts, nil
}

func (ts *targetCPUHost) Selected(c ICandidate) ITarget {
	ts.SetCurrent(ts.GetCurrent() + c.GetScore())
	return ts
}

func (ts *targetCPUHost) GetCPUCount() int {
	return int(ts.IHost.(*cpuHost).cpuCount)
}
