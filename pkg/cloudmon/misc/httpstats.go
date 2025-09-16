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

package misc

import (
	"fmt"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type SHttpStats struct {
	HttpCode2xx    float64 `json:"duration.2XX"`
	HttpCode4xx    float64 `json:"duration.4XX"`
	HttpCode5xx    float64 `json:"duration.5XX"`
	HitHttpCode2xx int64   `json:"hit.2XX"`
	HitHttpCode4xx int64   `json:"hit.4XX"`
	HitHttpCode5xx int64   `json:"hit.5XX"`

	Method string `json:"method"`
	Name   string `json:"name"`
	Path   string `json:"path"`
}

type sApiHttpStats struct {
	SHttpStats

	Paths []SHttpStats `json:"paths"`
}

func (apiStats *sApiHttpStats) convertSnapshot(now time.Time) *sHttpStatsSnapshot {
	snapshot := &sHttpStatsSnapshot{
		snapshotAt: now,
		stats:      map[string]*SHttpStats{},
	}
	apiStats.SHttpStats.Method = "any"
	apiStats.SHttpStats.Path = "any"
	apiStats.SHttpStats.Name = "any"
	snapshot.stats[getStatsKey(apiStats.SHttpStats.Method, apiStats.SHttpStats.Path, apiStats.SHttpStats.Name)] = &apiStats.SHttpStats
	for i := range apiStats.Paths {
		pathStats := apiStats.Paths[i]
		snapshot.stats[getStatsKey(pathStats.Method, pathStats.Path, pathStats.Name)] = &pathStats
	}
	return snapshot
}

type sHttpStatsExt struct {
	Duration2xx float64 `json:"duration_2xx"`
	Duration4xx float64 `json:"duration_4xx"`
	Duration5xx float64 `json:"duration_5xx"`
	Hit2xx      int64   `json:"hit_2xx"`
	Hit4xx      int64   `json:"hit_4xx"`
	Hit5xx      int64   `json:"hit_5xx"`

	Duration2xxDiff float64 `json:"duration_2xx_diff"`
	Duration4xxDiff float64 `json:"duration_4xx_diff"`
	Duration5xxDiff float64 `json:"duration_5xx_diff"`
	Hit2xxDiff      int64   `json:"hit_2xx_diff"`
	Hit4xxDiff      int64   `json:"hit_4xx_diff"`
	Hit5xxDiff      int64   `json:"hit_5xx_diff"`

	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`

	HasDiff bool `json:"has_diff"`
}

func (v sHttpStatsExt) DelayMs2xx() float64 {
	if v.Hit2xxDiff > 0 {
		return v.Duration2xxDiff / float64(v.Hit2xxDiff)
	}
	return -1
}

func (v sHttpStatsExt) Qps2xx(interval time.Duration) float64 {
	if interval > 0 {
		return float64(v.Hit2xxDiff) / interval.Seconds()
	}
	return -1
}

func (v sHttpStatsExt) DelayMs4xx() float64 {
	if v.Hit4xxDiff > 0 {
		return v.Duration4xxDiff / float64(v.Hit4xxDiff)
	}
	return -1
}

func (v sHttpStatsExt) Qps4xx(interval time.Duration) float64 {
	if interval > 0 {
		return float64(v.Hit4xxDiff) / interval.Seconds()
	}
	return -1
}

func (v sHttpStatsExt) DelayMs5xx() float64 {
	if v.Hit5xxDiff > 0 {
		return v.Duration5xxDiff / float64(v.Hit5xxDiff)
	}
	return -1
}

func (v sHttpStatsExt) Qps5xx(interval time.Duration) float64 {
	if interval > 0 {
		return float64(v.Hit5xxDiff) / interval.Seconds()
	}
	return -1
}

func (v sHttpStatsExt) Qps(interval time.Duration) float64 {
	if interval > 0 {
		return float64(v.Hit2xxDiff+v.Hit4xxDiff+v.Hit5xxDiff) / interval.Seconds()
	}
	return -1
}

func (v sHttpStatsExt) DelayMs() float64 {
	if v.HitDiff() > 0 {
		return v.DurationMsDiff() / float64(v.HitDiff())
	}
	return -1
}

func (v sHttpStatsExt) DurationMs() float64 {
	return v.Duration2xx + v.Duration4xx + v.Duration5xx
}

func (v sHttpStatsExt) Hit() int64 {
	return v.Hit2xx + v.Hit4xx + v.Hit5xx
}

func (v sHttpStatsExt) DurationMsDiff() float64 {
	return v.Duration2xxDiff + v.Duration4xxDiff + v.Duration5xxDiff
}

func (v sHttpStatsExt) HitDiff() int64 {
	return v.Hit2xxDiff + v.Hit4xxDiff + v.Hit5xxDiff
}

func (v sHttpStatsExt) Percent2xx() float64 {
	if v.DurationMsDiff() > 0 {
		return v.Duration2xxDiff * 100 / v.DurationMsDiff()
	}
	return -1
}

func (v sHttpStatsExt) PercentHit2xx() float64 {
	if v.HitDiff() > 0 {
		return float64(v.Hit2xxDiff) * 100 / float64(v.HitDiff())
	}
	return -1
}

func (v sHttpStatsExt) Percent4xx() float64 {
	if v.DurationMsDiff() > 0 {
		return v.Duration4xxDiff * 100 / v.DurationMsDiff()
	}
	return -1
}

func (v sHttpStatsExt) PercentHit4xx() float64 {
	if v.HitDiff() > 0 {
		return float64(v.Hit4xxDiff) * 100 / float64(v.HitDiff())
	}
	return -1
}

func (v sHttpStatsExt) Percent5xx() float64 {
	if v.DurationMsDiff() > 0 {
		return v.Duration5xxDiff * 100 / v.DurationMsDiff()
	}
	return -1
}

func (v sHttpStatsExt) PercentHit5xx() float64 {
	if v.HitDiff() > 0 {
		return float64(v.Hit5xxDiff) * 100 / float64(v.HitDiff())
	}
	return -1
}

type sHttpStatsSnapshot struct {
	snapshotAt time.Time
	stats      map[string]*SHttpStats
}

type sHttpStatsDiff struct {
	snapshotAt time.Time
	interval   time.Duration
	stats      map[string]*sHttpStatsExt
}

func calculateHttpStatsDiff(prevSnap *sHttpStatsSnapshot, nowSnap *sHttpStatsSnapshot) *sHttpStatsDiff {
	diff := sHttpStatsDiff{
		snapshotAt: nowSnap.snapshotAt,
		stats:      map[string]*sHttpStatsExt{},
	}
	if prevSnap != nil {
		rootKey := getStatsKey("any", "any", "any")
		prevRoot := prevSnap.stats[rootKey]
		nowRoot := nowSnap.stats[rootKey]

		if prevRoot.HttpCode2xx > nowRoot.HttpCode2xx || prevRoot.HttpCode4xx > nowRoot.HttpCode4xx || prevRoot.HttpCode5xx > nowRoot.HttpCode5xx ||
			prevRoot.HitHttpCode2xx > nowRoot.HitHttpCode2xx || prevRoot.HitHttpCode4xx > nowRoot.HitHttpCode4xx || prevRoot.HitHttpCode5xx > nowRoot.HitHttpCode5xx {
			// detect a reset, skip this round
			prevSnap = nil
		}
	}
	if prevSnap != nil {
		diff.interval = nowSnap.snapshotAt.Sub(prevSnap.snapshotAt)
	} else {
		intvMin := options.Options.CollectServiceMetricIntervalMinute
		if intvMin <= 0 {
			intvMin = 1
		}
		diff.interval = time.Duration(intvMin) * time.Minute
	}

	for k := range nowSnap.stats {
		v := nowSnap.stats[k]
		diffStats := sHttpStatsExt{
			Duration2xx: v.HttpCode2xx,
			Duration4xx: v.HttpCode4xx,
			Duration5xx: v.HttpCode5xx,
			Hit2xx:      v.HitHttpCode2xx,
			Hit4xx:      v.HitHttpCode4xx,
			Hit5xx:      v.HitHttpCode5xx,

			Method: v.Method,
			Path:   v.Path,
			Name:   v.Name,
		}
		if prevSnap != nil {
			if prevStats, ok := prevSnap.stats[k]; ok {
				diffStats.HasDiff = true
				diffStats.Duration2xxDiff = v.HttpCode2xx - prevStats.HttpCode2xx
				diffStats.Duration4xxDiff = v.HttpCode4xx - prevStats.HttpCode4xx
				diffStats.Duration5xxDiff = v.HttpCode5xx - prevStats.HttpCode5xx
				diffStats.Hit2xxDiff = v.HitHttpCode2xx - prevStats.HitHttpCode2xx
				diffStats.Hit4xxDiff = v.HitHttpCode4xx - prevStats.HitHttpCode4xx
				diffStats.Hit5xxDiff = v.HitHttpCode5xx - prevStats.HitHttpCode5xx
			}
		}

		diff.stats[k] = &diffStats
	}
	return &diff
}

var (
	httpStatsSnapshot = map[string]*sHttpStatsSnapshot{}
)

func getStatsKey(method, path, name string) string {
	return fmt.Sprintf("%s.%s.%s", method, path, name)
}

func getSnapshotKey(serviceName, url string) string {
	return fmt.Sprintf("%s.%s", serviceName, url)
}

func updateHttpStatsSnapshot(serviceName string, url string, now time.Time, apiStats sApiHttpStats, serviceType, regionId, version string) []influxdb.SMetricData {
	snapshot := apiStats.convertSnapshot(now)

	snapshotKey := getSnapshotKey(serviceName, url)
	var vdiffStats *sHttpStatsDiff
	if prevSnap, ok := httpStatsSnapshot[snapshotKey]; ok {
		vdiffStats = calculateHttpStatsDiff(prevSnap, snapshot)
	} else {
		// no prev records, just add
		vdiffStats = calculateHttpStatsDiff(nil, snapshot)
	}
	httpStatsSnapshot[snapshotKey] = snapshot

	metrics := vdiffStats.metrics(serviceName, serviceType, regionId, version)

	log.Debugf("updateHttpStatsSnapshot %s %s snapshotAt: %s diffAt: %s intval: %f metrics: %d", serviceName, url, snapshot.snapshotAt, vdiffStats.snapshotAt, vdiffStats.interval.Seconds(), len(metrics))

	return metrics
}

func appendMetric(metrics []influxdb.SKeyValue, key string, v float64) []influxdb.SKeyValue {
	if v >= 0 {
		metrics = append(metrics, influxdb.SKeyValue{
			Key:   key,
			Value: fmt.Sprintf("%f", v),
		})
	}
	return metrics
}

func (diff *sHttpStatsDiff) metrics(service, serviceType, regionId, version string) []influxdb.SMetricData {
	metrics := make([]influxdb.SMetricData, 0)
	genTags := func(v *sHttpStatsExt) []influxdb.SKeyValue {
		return []influxdb.SKeyValue{
			{
				Key:   "service",
				Value: service,
			},
			{
				Key:   "service_type",
				Value: serviceType,
			},
			{
				Key:   "region",
				Value: regionId,
			},
			{
				Key:   "version",
				Value: version,
			},
			{
				Key:   "method",
				Value: v.Method,
			},
			{
				Key:   "path",
				Value: v.Path,
			},
			{
				Key:   "name",
				Value: v.Name,
			},
			{
				Key:   "interval_secs",
				Value: fmt.Sprintf("%f", diff.interval.Seconds()),
			},
		}
	}
	for k := range diff.stats {
		v := diff.stats[k]

		// ignore stats with no hit
		if v.Hit() <= 0 {
			continue
		}
		// ignore stats with no diff
		if v.HasDiff && v.HitDiff() <= 0 {
			continue
		}

		metric := influxdb.SMetricData{
			Name:      METIRCY_TYPE_HTTP_REQUST,
			Timestamp: diff.snapshotAt,
			Tags:      genTags(v),
			Metrics: []influxdb.SKeyValue{
				{
					Key:   "duration_ms_any",
					Value: fmt.Sprintf("%f", v.DurationMs()),
				},
				{
					Key:   "hit_any",
					Value: fmt.Sprintf("%d", v.Hit()),
				},
			},
		}
		if v.HasDiff {
			metric.Metrics = appendMetric(metric.Metrics, "dura_ms_delta_any", v.DurationMsDiff())
			metric.Metrics = appendMetric(metric.Metrics, "hit_delta_any", float64(v.HitDiff()))
			metric.Metrics = appendMetric(metric.Metrics, "delay_ms_any", v.DelayMs())
			metric.Metrics = appendMetric(metric.Metrics, "qps_any", v.Qps(diff.interval))
		}
		metric.Metrics = append(metric.Metrics, []influxdb.SKeyValue{
			{
				Key:   "duration_ms_2xx",
				Value: fmt.Sprintf("%f", v.Duration2xx),
			},
			{
				Key:   "hit_2xx",
				Value: fmt.Sprintf("%d", v.Hit2xx),
			},
		}...)
		if v.HasDiff {
			metric.Metrics = appendMetric(metric.Metrics, "dura_ms_delta_2xx", v.Duration2xxDiff)
			metric.Metrics = appendMetric(metric.Metrics, "hit_delta_2xx", float64(v.Hit2xxDiff))
			metric.Metrics = appendMetric(metric.Metrics, "delay_ms_2xx", v.DelayMs2xx())
			metric.Metrics = appendMetric(metric.Metrics, "percent_hit_2xx", v.PercentHit2xx())
			metric.Metrics = appendMetric(metric.Metrics, "percent_duration_2xx", v.Percent2xx())
			metric.Metrics = appendMetric(metric.Metrics, "qps_2xx", v.Qps2xx(diff.interval))
		}
		metric.Metrics = append(metric.Metrics, []influxdb.SKeyValue{
			{
				Key:   "duration_ms_4xx",
				Value: fmt.Sprintf("%f", v.Duration4xx),
			},
			{
				Key:   "hit_4xx",
				Value: fmt.Sprintf("%d", v.Hit4xx),
			},
		}...)
		if v.HasDiff {
			metric.Metrics = appendMetric(metric.Metrics, "dura_ms_delta_4xx", v.Duration4xxDiff)
			metric.Metrics = appendMetric(metric.Metrics, "hit_delta_4xx", float64(v.Hit4xxDiff))
			metric.Metrics = appendMetric(metric.Metrics, "delay_ms_4xx", v.DelayMs4xx())
			metric.Metrics = appendMetric(metric.Metrics, "percent_hit_4xx", v.PercentHit4xx())
			metric.Metrics = appendMetric(metric.Metrics, "percent_duration_4xx", v.Percent4xx())
			metric.Metrics = appendMetric(metric.Metrics, "qps_4xx", v.Qps4xx(diff.interval))
		}
		metric.Metrics = append(metric.Metrics, []influxdb.SKeyValue{
			{
				Key:   "duration_ms_5xx",
				Value: fmt.Sprintf("%f", v.Duration5xx),
			},
			{
				Key:   "hit_5xx",
				Value: fmt.Sprintf("%d", v.Hit5xx),
			},
		}...)
		if v.HasDiff {
			metric.Metrics = appendMetric(metric.Metrics, "dura_ms_delta_5xx", v.Duration5xxDiff)
			metric.Metrics = appendMetric(metric.Metrics, "hit_delta_5xx", float64(v.Hit5xxDiff))
			metric.Metrics = appendMetric(metric.Metrics, "delay_ms_5xx", v.DelayMs5xx())
			metric.Metrics = appendMetric(metric.Metrics, "percent_hit_5xx", v.PercentHit5xx())
			metric.Metrics = appendMetric(metric.Metrics, "percent_duration_5xx", v.Percent5xx())
			metric.Metrics = appendMetric(metric.Metrics, "qps_5xx", v.Qps5xx(diff.interval))
		}

		metrics = append(metrics, metric)
	}
	return metrics
}
