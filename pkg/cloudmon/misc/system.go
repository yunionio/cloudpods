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
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

const (
	SYSTEM_METRIC_DATABASE = "system"

	METIRCY_TYPE_HTTP_REQUST = "http_request"
	METIRCY_TYPE_WORKER      = "worker"
	METIRCY_TYPE_DB_STATS    = "db_stats"
	METIRCY_TYPE_PROCESS     = "process"
)

func getEndpoints(s *mcclient.ClientSession) ([]api.EndpointDetails, error) {
	ret := make([]api.EndpointDetails, 0)
	params := baseoptions.BaseListOptions{}
	limit := 1024
	params.Limit = &limit
	boolTrue := true
	params.Details = &boolTrue
	params.Scope = "system"
	params.Filter = []string{
		"interface.equals(internal)",
		"enabled.equals(1)",
	}

	for {
		offset := len(ret)
		params.Offset = &offset
		resp, err := identity.EndpointsV3.List(s, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrapf(err, "Endpoints.List")
		}
		for i := range resp.Data {
			endpoint := api.EndpointDetails{}
			err := resp.Data[i].Unmarshal(&endpoint)
			if err != nil {
				return nil, errors.Wrapf(err, "Unmarshal")
			}
			ret = append(ret, endpoint)
		}
		if len(ret) >= resp.Total {
			break
		}
	}
	return ret, nil
}

func getHosts(s *mcclient.ClientSession) ([]compute_api.HostDetails, error) {
	params := compute_api.HostListInput{}
	boolFalse := false
	limit := 100
	params.Limit = &limit
	params.Brand = []string{compute_api.CLOUD_PROVIDER_ONECLOUD}
	params.Scope = "system"
	params.Status = []string{compute_api.HOST_STATUS_RUNNING}
	params.HostStatus = []string{compute_api.HOST_ONLINE}
	params.Details = &boolFalse

	hosts := []compute_api.HostDetails{}
	for {
		offset := len(hosts)
		params.Offset = &offset
		resp, err := compute.Hosts.List(s, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrapf(err, "Hosts.List")
		}
		for i := range resp.Data {
			host := compute_api.HostDetails{}
			err := resp.Data[i].Unmarshal(&host)
			if err != nil {
				return nil, errors.Wrapf(err, "Unmarshal")
			}
			hosts = append(hosts, host)
		}
		if len(hosts) >= resp.Total {
			break
		}
	}
	return hosts, nil
}

func CollectServiceMetrics(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if options.Options.DisableServiceMetric {
		return
	}
	s := auth.GetAdminSession(ctx, options.Options.CommonOptions.Region)
	urls, err := tsdb.GetDefaultServiceSourceURLs(s, options.Options.SessionEndpointType)
	if err != nil {
		return
	}

	tk := auth.AdminCredential().GetTokenString()
	err = func() error {
		endpoints, err := getEndpoints(s)
		if err != nil {
			return errors.Wrapf(err, "getEndpoints")
		}
		metrics := []influxdb.SMetricData{}
		for _, ep := range endpoints {
			if utils.IsInStringArray(ep.ServiceType, apis.EXTERNAL_SERVICES) {
				continue
			}
			url := httputils.JoinPath(ep.Url, "version")
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
			version, _ := io.ReadAll(resp.Body)
			if len(version) == 0 {
				continue
			}

			part, err := collectServiceMetrics(ctx, ep, string(version), tk)
			if err != nil {
				log.Errorf("collect service %s metric error: %v", ep.ServiceType, err)
			}
			metrics = append(metrics, part...)
		}
		if len(metrics) > 0 {
			err := influxdb.SendMetrics(urls, SYSTEM_METRIC_DATABASE, metrics, false)
			if err != nil {
				return errors.Wrapf(err, "SendMetrics")
			}
		}
		return nil
	}()
	if err != nil {
		log.Errorf("collect service metric error: %v", err)
	}

	{
		hosts, err := getHosts(s)
		if err != nil {
			log.Errorf("get hosts error: %v", err)
		}
		metrics := []influxdb.SMetricData{}
		for _, host := range hosts {
			part := collectHostMetrics(ctx, host, tk)
			metrics = append(metrics, part...)
		}
		if len(metrics) > 0 {
			err := influxdb.SendMetrics(urls, SYSTEM_METRIC_DATABASE, metrics, false)
			if err != nil {
				log.Errorf("send host metrics error: %v", err)
			}
		}
	}
}

func collectApiStatsMetrics(ctx context.Context, serviceName string, serviceType string, regionId string, url string, version, token string) ([]influxdb.SMetricData, error) {
	if len(url) == 0 || utils.IsInStringArray(serviceType, []string{"mcp-server"}) {
		return []influxdb.SMetricData{}, nil
	}
	log.Debugf("collectApiStatsMetrics %s %s %s %s %s", serviceName, serviceType, regionId, url, version)
	statsUrl := httputils.JoinPath(baseUrlF(url), "stats")
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

	stats := sApiHttpStats{}
	err = ret.Unmarshal(&stats)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}

	metrics := updateHttpStatsSnapshot(serviceName, url, time.Now(), stats, serviceType, regionId, version)

	return metrics, nil
}

func collectWorkerMetrics(ctx context.Context, url, service, serviceType, regionId, version, token string) ([]influxdb.SMetricData, error) {
	if len(url) == 0 || utils.IsInStringArray(serviceType, []string{"mcp-server"}) {
		return []influxdb.SMetricData{}, nil
	}
	statsUrl := httputils.JoinPath(baseUrlF(url), "worker_stats")
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
				{
					Key:   "total_workload",
					Value: fmt.Sprintf("%d", worker.ActiveWorkerCnt+worker.QueueCnt+worker.DetachWorkerCnt),
				},
				{
					Key:   "active_workload",
					Value: fmt.Sprintf("%d", worker.ActiveWorkerCnt+worker.DetachWorkerCnt),
				},
			},
		}
		result = append(result, metric)

	}
	return result, nil
}

func collectDatabaseMetrics(ctx context.Context, ep api.EndpointDetails, version, token string) ([]influxdb.SMetricData, error) {
	if utils.IsInStringArray(ep.ServiceType, []string{"cloudmon", "webconsole", "k8s", "vpcagent", "yunionapi", "yunionagent"}) {
		return []influxdb.SMetricData{}, nil
	}
	statsUrl := httputils.JoinPath(baseUrlF(ep.Url), "db_stats")
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
	if utils.IsInStringArray(ep.ServiceType, []string{"monitor", "webconsole", "k8s", "mcp-server"}) {
		return []influxdb.SMetricData{}, nil
	}
	statsUrl := httputils.JoinPath(baseUrlF(ep.Url), "process_stats")
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

func baseUrlF(baseurl string) string {
	obj, _ := url.Parse(baseurl)
	lastSlashPos := strings.LastIndex(obj.Path, "/")
	if lastSlashPos >= 0 {
		lastSeg := obj.Path[lastSlashPos+1:]
		verReg := regexp.MustCompile(`^v\d+`)
		if verReg.MatchString(lastSeg) {
			obj.Path = obj.Path[:lastSlashPos]
		}
	}
	ret := obj.String()
	return ret
}

func collectServiceMetrics(ctx context.Context, ep api.EndpointDetails, version, token string) ([]influxdb.SMetricData, error) {
	ret, errs := []influxdb.SMetricData{}, []error{}
	stats, err := collectApiStatsMetrics(ctx, ep.ServiceName, ep.ServiceType, ep.RegionId, ep.Url, version, token)
	if err != nil {
		errs = append(errs, err)
	}
	ret = append(ret, stats...)
	worker, err := collectWorkerMetrics(ctx, ep.Url, ep.ServiceName, ep.ServiceType, ep.RegionId, version, token)
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

func collectHostMetrics(ctx context.Context, host compute_api.HostDetails, token string) []influxdb.SMetricData {
	metrics := []influxdb.SMetricData{}
	service := fmt.Sprintf("host-%s", host.Name)
	part, err := collectWorkerMetrics(ctx, host.ManagerUri, service, "host", host.Region, host.Version, token)
	if err != nil {
		log.Errorf("collect host %s metric error: %v", service, err)
	} else {
		metrics = append(metrics, part...)
	}
	part, err = collectApiStatsMetrics(ctx, service, "host", host.Region, host.ManagerUri, host.Version, token)
	if err != nil {
		log.Errorf("collect host %s metric error: %v", service, err)
	} else {
		metrics = append(metrics, part...)
	}
	return metrics
}
