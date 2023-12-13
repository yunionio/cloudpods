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

type memAvailable struct{}

func newMemAvailable() IMetricDriver {
	return &memAvailable{}
}

func (m *memAvailable) GetType() monitor.MigrationAlertMetricType {
	return monitor.MigrationAlertMetricTypeMemAvailable
}

func (ma *memAvailable) GetTsdbQuery() *TsdbQuery {
	return &TsdbQuery{
		Database:    monitor.METRIC_DATABASE_TELE,
		Measurement: "mem",
		Fields:      []string{"total", "free", "available"},
	}
}

func (ma *memAvailable) GetCandidate(obj jsonutils.JSONObject, host IHost, _ *tsdb.DataSource) (ICandidate, error) {
	return newMemCandidate(obj, host)
}

func (ma *memAvailable) SetHostCurrent(host IHost, vals map[string]float64) error {
	return setHostCurrent(host, vals, "free")
}

func (ma *memAvailable) GetTarget(host jsonutils.JSONObject) (ITarget, error) {
	return newTargetMemHost(host)
}

func (ma *memAvailable) GetCondition(s *monitor.AlertSetting) (ICondition, error) {
	t, err := GetAlertSettingThreshold(s)
	if err != nil {
		return nil, errors.Wrap(err, "GetAlertSettingThreshold")
	}
	return newMemoryCond(t), nil
}

// memCondition implements ICondition
type memCondition struct {
	value float64
}

func newMemoryCond(value float64) ICondition {
	return &memCondition{
		value: value,
	}
}

func (m *memCondition) GetThreshold() float64 {
	return m.value
}

func (m *memCondition) GetSourceThresholdDelta(threshold float64, host IHost) float64 {
	// mem.available
	return threshold - host.GetCurrent()
}

func (m *memCondition) IsFitTarget(settings *monitor.MigrationAlertSettings, t ITarget, c ICandidate) error {
	tScore := t.GetCurrent() - c.GetScore()
	gtThreshold := tScore > m.GetThreshold()
	if gtThreshold {
		return nil
	}

	// 1G = 1 * 1024 * 1024 * 1024 (bytes)
	MAX_THRESHOLD := float64(1 * 1024 * 1024 * 1024)
	src := settings.Source
	srcHostIds := []string{}
	if src != nil {
		srcHostIds = src.HostIds
	}
	// only when srcHostIds isn't empty
	if len(srcHostIds) != 0 {
		if !gtThreshold && !sets.NewString(srcHostIds...).Has(t.GetId()) && tScore > MAX_THRESHOLD {
			// if target host is not in source specified hosts and calculated score is greater than MAX_THRESHOLD
			log.Infof("let host:%s:current(%f) + guest:%s:score(%f) > MAX_THRESHOLD(%f) to fit target, because it's not in source specified hosts", t.GetName(), t.GetCurrent(), c.GetName(), c.GetScore(), MAX_THRESHOLD)
			return nil
		}
	}

	return errors.Errorf("host:%s:current(%f) - guest:%s:score(%f) <= threshold(%f)", t.GetName(), t.GetCurrent(), c.GetName(), c.GetScore(), m.GetThreshold())
}

// memCandidate implements ICandidate
type memCandidate struct {
	*guestResource
	score float64
}

func newMemCandidate(gst jsonutils.JSONObject, host IHost) (ICandidate, error) {
	res, err := newGuestResource(gst, host.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "newGuestResource")
	}

	memSizeMB, err := gst.Int("vmem_size")
	if err != nil {
		return nil, errors.Wrap(err, "get vmem_size")
	}

	/* unit of influxdb query is byte
	> select free, available, total from mem where host_id = 'eda7c6f5-f714-4d59-8d6a-16b658712b07' limit 1;
	name: mem
	time                 free        available   total
	----                 ----        ---------   -----
	2022-05-02T00:00:00Z 15399550976 94193070080 270276599808
	*/

	return &memCandidate{
		guestResource: res,
		score:         float64(memSizeMB * 1024 * 1024),
	}, nil
}

func (m *memCandidate) GetScore() float64 {
	return m.score
}

type memHost struct {
	*HostResource
	availableMemSize float64
}

func newMemHost(obj jsonutils.JSONObject) (IHost, error) {
	host, err := newHostResource(obj)
	if err != nil {
		return nil, errors.Wrap(err, "newHostResource")
	}
	return &memHost{
		HostResource: host,
	}, nil
}

func (ts *memHost) GetCurrent() float64 {
	return ts.availableMemSize
}

func (ts *memHost) SetCurrent(val float64) IHost {
	ts.availableMemSize = val
	return ts
}

func (ts *memHost) Compare(oh IHost) bool {
	return ts.GetCurrent() > oh.GetCurrent()
}

type targetMemHost struct {
	IHost
}

func newTargetMemHost(obj jsonutils.JSONObject) (ITarget, error) {
	host, err := newMemHost(obj)
	if err != nil {
		return nil, errors.Wrap(err, "newMemHost")
	}
	ts := &targetMemHost{
		IHost: host,
	}
	return ts, nil
}

func (ts *targetMemHost) Selected(c ICandidate) ITarget {
	ts.SetCurrent(ts.GetCurrent() - c.GetScore())
	return ts
}
