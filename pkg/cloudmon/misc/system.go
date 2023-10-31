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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

const (
	SYSTEM_METRIC_DATABASE = "system"

	METIRCY_TYPE_HTTP_REQUST = "http_request"
	METIRCY_TYPE_WORKER      = "worker"
	METIRCY_TYPE_DB_STATS    = "db_stats"
	METIRCY_TYPE_PROCESS     = "process"
)

func getEndpoints(ctx context.Context, s *mcclient.ClientSession) ([]api.EndpointDetails, error) {
	resp, err := identity.EndpointsV3.List(s, jsonutils.Marshal(map[string]string{
		"scope":     "system",
		"enable":    "true",
		"details":   "true",
		"interface": "internal",
		"limit":     "50",
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "Endpoints.List")
	}
	ret := []api.EndpointDetails{}
	return ret, jsonutils.Update(&ret, resp.Data)
}

func CollectServiceMetrics(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if options.Options.DisableServiceMetric {
		return
	}
	s := auth.GetAdminSession(ctx, options.Options.CommonOptions.Region)
	err := func() error {
		endpoints, err := getEndpoints(ctx, s)
		if err != nil {
			return errors.Wrapf(err, "getEndpoints")
		}
		metrics := []influxdb.SMetricData{}
		for _, ep := range endpoints {
			if utils.IsInStringArray(ep.ServiceType, apis.NO_RESOURCE_SERVICES) || ep.ServiceType == apis.SERVICE_TYPE_IMAGE {
				continue
			}
			url := httputils.JoinPath(ep.Url, "version")
			tk := auth.AdminCredential().GetTokenString()
			hdr := http.Header{}
			hdr.Set("X-Auth-Token", tk)
			resp, err := httputils.Request(
				httputils.GetDefaultClient(),
				ctx, "GET",
				url, hdr, nil, false,
			)
			if err != nil {
				continue
			}
			defer resp.Body.Close()
			version, _ := ioutil.ReadAll(resp.Body)
			if len(version) == 0 {
				continue
			}

			part, err := collectServiceMetrics(ctx, ep, string(version), tk)
			if err != nil {
				log.Errorf("collect service %s metric error: %v", ep.ServiceType, err)
			}
			metrics = append(metrics, part...)
		}
		urls, err := tsdb.GetDefaultServiceSourceURLs(s, options.Options.SessionEndpointType)
		if err != nil {
			return errors.Wrap(err, "GetServiceURLs")
		}
		return influxdb.SendMetrics(urls, SYSTEM_METRIC_DATABASE, metrics, false)
	}()
	if err != nil {
		log.Errorf("collect service metric error: %v", err)
	}
}

func collectStatsMetrics(ctx context.Context, ep api.EndpointDetails, version, token string) ([]influxdb.SMetricData, error) {
	statsUrl := httputils.JoinPath(ep.Url, "stats")
	hdr := http.Header{}
	hdr.Set("X-Auth-Token", token)
	_, ret, err := httputils.JSONRequest(
		httputils.GetDefaultClient(),
		ctx, "GET",
		statsUrl, hdr, nil, false,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "request")
	}

	if gotypes.IsNil(ret) {
		return []influxdb.SMetricData{}, nil
	}

	stats := struct {
		HttpCode2xx    float64 `json:"duration.2XX"`
		HttpCode4xx    float64 `json:"duration.4XX"`
		HttpCode5xx    float64 `json:"duration.5XX"`
		HitHttpCode2xx int     `json:"hit.2XX"`
		HitHttpCode4xx int     `json:"hit.4XX"`
		HitHttpCode5xx int     `json:"hit.5XX"`
		Paths          []struct {
			HttpCode2xx    float64 `json:"duration.2XX"`
			HttpCode4xx    float64 `json:"duration.4XX"`
			HttpCode5xx    float64 `json:"duration.5XX"`
			HitHttpCode2xx int     `json:"hit.2XX"`
			HitHttpCode4xx int     `json:"hit.4XX"`
			HitHttpCode5xx int     `json:"hit.5XX"`
			Method         string
			Name           string
			Path           string
		} `json:"paths"`
	}{}
	err = ret.Unmarshal(&stats)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	result := []influxdb.SMetricData{}
	metric := influxdb.SMetricData{
		Name:      METIRCY_TYPE_HTTP_REQUST,
		Timestamp: time.Now(),
		Tags: []influxdb.SKeyValue{
			{
				Key:   "version",
				Value: version,
			},
			{
				Key:   "service",
				Value: ep.ServiceName,
			},
		},
		Metrics: []influxdb.SKeyValue{
			{
				Key:   "duration.2xx",
				Value: fmt.Sprintf("%.2f", stats.HttpCode2xx),
			},
			{
				Key:   "duration.4xx",
				Value: fmt.Sprintf("%.2f", stats.HttpCode4xx),
			},
			{
				Key:   "duration.5xx",
				Value: fmt.Sprintf("%.2f", stats.HttpCode5xx),
			},
			{
				Key:   "hit.2xx",
				Value: fmt.Sprintf("%d", stats.HitHttpCode2xx),
			},
			{
				Key:   "hit.4xx",
				Value: fmt.Sprintf("%d", stats.HitHttpCode4xx),
			},
			{
				Key:   "hit.5xx",
				Value: fmt.Sprintf("%d", stats.HitHttpCode5xx),
			},
		},
	}
	result = append(result, metric)
	for _, path := range stats.Paths {
		metric = influxdb.SMetricData{
			Name:      METIRCY_TYPE_HTTP_REQUST,
			Timestamp: time.Now(),
			Tags: []influxdb.SKeyValue{
				{
					Key:   "version",
					Value: version,
				},
				{
					Key:   "service",
					Value: ep.ServiceName,
				},
				{
					Key:   "method",
					Value: path.Method,
				},
				{
					Key:   "path",
					Value: path.Path,
				},
				{
					Key:   "url",
					Value: path.Name,
				},
			},
			Metrics: []influxdb.SKeyValue{
				{
					Key:   "duration.2xx",
					Value: fmt.Sprintf("%.2f", path.HttpCode2xx),
				},
				{
					Key:   "duration.4xx",
					Value: fmt.Sprintf("%.2f", path.HttpCode4xx),
				},
				{
					Key:   "duration.5xx",
					Value: fmt.Sprintf("%.2f", path.HttpCode5xx),
				},
				{
					Key:   "hit.2xx",
					Value: fmt.Sprintf("%d", path.HitHttpCode2xx),
				},
				{
					Key:   "hit.4xx",
					Value: fmt.Sprintf("%d", path.HitHttpCode4xx),
				},
				{
					Key:   "hit.5xx",
					Value: fmt.Sprintf("%d", path.HitHttpCode5xx),
				},
			},
		}
		result = append(result, metric)
	}
	return result, nil
}

func collectWorkerMetrics(ctx context.Context, ep api.EndpointDetails, version, token string) ([]influxdb.SMetricData, error) {
	statsUrl := httputils.JoinPath(ep.Url, "worker_stats")
	hdr := http.Header{}
	hdr.Set("X-Auth-Token", token)
	_, ret, err := httputils.JSONRequest(
		httputils.GetDefaultClient(),
		ctx, "GET",
		statsUrl, hdr, nil, false,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "request")
	}

	if gotypes.IsNil(ret) {
		return []influxdb.SMetricData{}, nil
	}

	workers := []struct {
		ActiveWorkerCnt int    `json:"active_worker_cnt"`
		AllowOverflow   bool   `json:"allow_overflow"`
		Backlog         int    `json:"backlog"`
		DbWorker        bool   `json:"db_worker"`
		DetachWorkerCnt int    `json:"detach_worker_cnt"`
		MaxWorkerCnt    int    `json:"max_worker_cnt"`
		Name            string `json:"name"`
		QueueCnt        int    `json:"queue_cnt"`
	}{}
	err = ret.Unmarshal(&workers, "workers")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}

	result := []influxdb.SMetricData{}
	for _, worker := range workers {
		metric := influxdb.SMetricData{
			Name:      METIRCY_TYPE_WORKER,
			Timestamp: time.Now(),
			Tags: []influxdb.SKeyValue{
				{
					Key:   "version",
					Value: version,
				},
				{
					Key:   "service",
					Value: ep.ServiceName,
				},
				{
					Key:   "worker_name",
					Value: worker.Name,
				},
			},
			Metrics: []influxdb.SKeyValue{
				{
					Key:   "active_worker_cnt",
					Value: fmt.Sprintf("%d", worker.ActiveWorkerCnt),
				},
				{
					Key:   "max_worker_cnt",
					Value: fmt.Sprintf("%d", worker.MaxWorkerCnt),
				},
				{
					Key:   "detach_worker_cnt",
					Value: fmt.Sprintf("%d", worker.DetachWorkerCnt),
				},
				{
					Key:   "queue_cnt",
					Value: fmt.Sprintf("%d", worker.QueueCnt),
				},
			},
		}
		result = append(result, metric)

	}
	return result, nil
}

func collectDatabaseMetrics(ctx context.Context, ep api.EndpointDetails, version, token string) ([]influxdb.SMetricData, error) {
	statsUrl := httputils.JoinPath(ep.Url, "db_stats")
	hdr := http.Header{}
	hdr.Set("X-Auth-Token", token)
	_, ret, err := httputils.JSONRequest(
		httputils.GetDefaultClient(),
		ctx, "GET",
		statsUrl, hdr, nil, false,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "request")
	}

	if gotypes.IsNil(ret) {
		return []influxdb.SMetricData{}, nil
	}

	stats := struct {
		Idle               int `json:"idle"`
		InUse              int `json:"in_use"`
		MaxIdleClosed      int `json:"max_idle_closed"`
		MaxIdleTimeClosed  int `json:"max_idle_time_closed"`
		MaxLifetimeClosed  int `json:"max_lifetime_closed"`
		MaxOpenConnections int `json:"max_open_connections"`
		OpenConnections    int `json:"open_connections"`
		WaitCount          int `json:"wait_count"`
		WaitDuration       int `json:"wait_duration"`
	}{}
	err = ret.Unmarshal(&stats, "db_stats")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	metric := influxdb.SMetricData{
		Name:      METIRCY_TYPE_DB_STATS,
		Timestamp: time.Now(),
		Tags: []influxdb.SKeyValue{
			{
				Key:   "version",
				Value: version,
			},
			{
				Key:   "service",
				Value: ep.ServiceName,
			},
		},
		Metrics: []influxdb.SKeyValue{
			{
				Key:   "idle",
				Value: fmt.Sprintf("%d", stats.Idle),
			},
			{
				Key:   "in_use",
				Value: fmt.Sprintf("%d", stats.InUse),
			},
			{
				Key:   "max_idle_closed",
				Value: fmt.Sprintf("%d", stats.MaxIdleClosed),
			},
			{
				Key:   "max_idle_time_closed",
				Value: fmt.Sprintf("%d", stats.MaxIdleTimeClosed),
			},
			{
				Key:   "max_lifetime_closed",
				Value: fmt.Sprintf("%d", stats.MaxLifetimeClosed),
			},
			{
				Key:   "max_open_connections",
				Value: fmt.Sprintf("%d", stats.MaxOpenConnections),
			},
			{
				Key:   "open_connections",
				Value: fmt.Sprintf("%d", stats.OpenConnections),
			},
			{
				Key:   "wait_count",
				Value: fmt.Sprintf("%d", stats.WaitCount),
			},
			{
				Key:   "wait_duration",
				Value: fmt.Sprintf("%d", stats.WaitDuration),
			},
		},
	}
	return []influxdb.SMetricData{metric}, nil
}

func collectProcessMetrics(ctx context.Context, ep api.EndpointDetails, version, token string) ([]influxdb.SMetricData, error) {
	statsUrl := httputils.JoinPath(ep.Url, "process_stats")
	hdr := http.Header{}
	hdr.Set("X-Auth-Token", token)
	_, ret, err := httputils.JSONRequest(
		httputils.GetDefaultClient(),
		ctx, "GET",
		statsUrl, hdr, nil, false,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "request")
	}

	if gotypes.IsNil(ret) {
		return []influxdb.SMetricData{}, nil
	}

	process := apis.ProcessStats{}
	err = ret.Unmarshal(&process, "process_stats")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	metric := influxdb.SMetricData{
		Name:      METIRCY_TYPE_PROCESS,
		Timestamp: time.Now(),
		Tags: []influxdb.SKeyValue{
			{
				Key:   "version",
				Value: version,
			},
			{
				Key:   "service",
				Value: ep.ServiceName,
			},
		},
		Metrics: []influxdb.SKeyValue{
			{
				Key:   "cpu_percent",
				Value: fmt.Sprintf("%.2f", process.CpuPercent),
			},
			{
				Key:   "mem_percent",
				Value: fmt.Sprintf("%.2f", process.MemPercent),
			},
			{
				Key:   "mem_size",
				Value: fmt.Sprintf("%d", process.MemSize),
			},
			{
				Key:   "goroutine_num",
				Value: fmt.Sprintf("%d", process.GoroutineNum),
			},
		},
	}
	return []influxdb.SMetricData{metric}, nil
}

func collectServiceMetrics(ctx context.Context, ep api.EndpointDetails, version, token string) ([]influxdb.SMetricData, error) {
	ret, errs := []influxdb.SMetricData{}, []error{}
	stats, err := collectStatsMetrics(ctx, ep, version, token)
	if err != nil {
		errs = append(errs, err)
	}
	ret = append(ret, stats...)
	worker, err := collectWorkerMetrics(ctx, ep, version, token)
	if err != nil {
		errs = append(errs, err)
	}
	ret = append(ret, worker...)
	db, err := collectDatabaseMetrics(ctx, ep, version, token)
	if err != nil {
		errs = append(errs, err)
	}
	ret = append(ret, db...)
	process, err := collectProcessMetrics(ctx, ep, version, token)
	if err != nil {
		errs = append(errs, err)
	}
	ret = append(ret, process...)
	return ret, errors.NewAggregate(errs)
}
