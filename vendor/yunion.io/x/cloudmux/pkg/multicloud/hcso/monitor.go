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

package hcso

import (
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/modules"
)

type MetricData struct {
	Namespace  string
	MetricName string
	Dimensions []struct {
		Name  string
		Value string
	}
	Datapoints []struct {
		Average   float64
		Timestamp int64
	}
	Unit string
}

func (r *SRegion) GetMetrics() ([]modules.SMetricMeta, error) {
	return r.ecsClient.CloudEye.ListMetrics()
}

func (r *SRegion) GetMetricsData(metrics []modules.SMetricMeta, since time.Time, until time.Time) ([]cloudprovider.MetricValues, error) {
	return r.client.getServerMetrics(&cloudprovider.MetricListOptions{ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_SERVER, MetricType: cloudprovider.VM_METRIC_TYPE_CPU_USAGE})
}

func (self *SHuaweiClient) getModelartsPoolMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	resp, err := self.modelartsPoolMonitor(opts.ResourceId, nil)
	if err != nil {
		return nil, err
	}
	metricData := []SModelartsMetric{}
	err = resp.Unmarshal(&metricData, "metrics")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	result := []cloudprovider.MetricValues{}
	for i := range metricData {
		isMB := false
		if metricData[i].Datapoints[0].Unit == "Megabytes" {
			isMB = true
			metricData[i].Datapoints[0].Unit = "Bytes"
		}
		ret := cloudprovider.MetricValues{
			Id:     opts.ResourceId,
			Unit:   metricData[i].Datapoints[0].Unit,
			Values: []cloudprovider.MetricValue{},
		}
		tags := map[string]string{}
		switch metricData[i].Metric.MetricName {
		case "cpuUsage":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_CPU_USAGE
		case "memUsedRate":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_MEM_USAGE
		case "gpuUtil":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_GPU_UTIL
		case "gpuMemUsage":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_GPU_MEM_USAGE
		case "npuUtil":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_NPU_UTIL
		case "npuMemUsage":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_NPU_MEM_USAGE
		case "diskAvailableCapacity":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_DISK_AVAILABLE_CAPACITY
		case "diskCapacity":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_DISK_CAPACITY
		case "diskUsedRate":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_DISK_USAGE
		default:
			log.Warningf("invalid metricName %s for %s %s", metricData[i].Metric.MetricName, opts.ResourceType, opts.ResourceId)
			continue
		}
		for _, value := range metricData[i].Datapoints {
			if isMB {
				value.Statistics[0].Value *= 1024
			}
			if value.Statistics[0].Value == -1 {
				value.Statistics[0].Value = 0
			}
			metricValue := cloudprovider.MetricValue{
				Value:     value.Statistics[0].Value,
				Timestamp: time.UnixMilli(value.Timestamp),
				Tags:      tags,
			}
			ret.Values = append(ret.Values, metricValue)
		}
		result = append(result, ret)
	}
	return result, nil
}

func (self *SHuaweiClient) getServerMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	result := []cloudprovider.MetricValues{}
	namespace, dimesionName, metricNames := "SYS.ECS", "instance_id", []string{
		"cpu_util",
		"mem_util",
		"disk_util_inband",
		"network_incoming_bytes_aggregate_rate",
		"network_outgoing_bytes_aggregate_rate",
		"disk_read_bytes_rate",
		"disk_write_bytes_rate",
		"disk_read_requests_rate",
		"disk_write_requests_rate",
	}
	for _, metricName := range metricNames {
		temp := make(map[string]string)
		temp["namespace"] = namespace
		temp["metric_name"] = metricName
		temp["from"] = strconv.Itoa(int(opts.StartTime.UnixMilli()))
		temp["to"] = strconv.Itoa(int(opts.EndTime.UnixMilli()))
		temp["period"] = "1"
		temp["filter"] = "average"
		temp["dim.0"] = fmt.Sprintf("%s,%s", dimesionName, opts.ResourceId)
		resp, err := self.commonMonitor(temp)
		if err != nil {
			log.Errorf("get monitor err:%s,input:%v", err.Error(), jsonutils.Marshal(temp))
			continue
		}
		metricData := MetricData{}
		err = resp.Unmarshal(&metricData)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		ret := cloudprovider.MetricValues{
			Id:     opts.ResourceId,
			Unit:   metricData.Unit,
			Values: []cloudprovider.MetricValue{},
		}
		tags := map[string]string{}
		switch metricData.MetricName {
		case "cpu_util":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_CPU_USAGE
		case "mem_util":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_MEM_USAGE
		case "disk_util_inband":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_USAGE
		case "network_incoming_bytes_aggregate_rate":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_NET_BPS_RX
			tags = map[string]string{"net_type": "internet"}
		case "network_outgoing_bytes_aggregate_rate":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_NET_BPS_TX
			tags = map[string]string{"net_type": "internet"}
		case "disk_read_bytes_rate":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS
		case "disk_write_bytes_rate":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS
		case "disk_read_requests_rate":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS
		case "disk_write_requests_rate":
			ret.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS
		default:
			log.Warningf("invalid metricName %s for %s %s", metricData.MetricName, opts.ResourceType, opts.ResourceId)
			continue
		}
		for _, value := range metricData.Datapoints {
			metricValue := cloudprovider.MetricValue{
				Value:     value.Average,
				Timestamp: time.UnixMilli(value.Timestamp),
				Tags:      tags,
			}
			ret.Values = append(ret.Values, metricValue)
		}
		result = append(result, ret)
	}
	return result, nil
}

func (self *SHuaweiClient) getRdsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	result := []cloudprovider.MetricValues{}
	namespace, dimesionName, metricNames := "SYS.RDS", "rds_cluster_id", []string{
		"rds001_cpu_util",
		"rds002_mem_util",
		"rds004_bytes_in",
		"rds004_bytes_in",
		"rds005_bytes_out",
		"rds039_disk_util",
		"rds049_disk_read_throughput",
		"rds050_disk_write_throughput",
		"rds006_conn_count",
		"rds008_qps",
		"rds009_tps",
		"rds013_innodb_reads",
		"rds014_innodb_writes",
	}
	switch opts.Engine {
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		dimesionName = "postgresql_cluster_id"
	case api.DBINSTANCE_TYPE_SQLSERVER:
		dimesionName = "rds_cluster_sqlserver_id"
	}

	for _, metricName := range metricNames {
		temp := make(map[string]string)
		temp["namespace"] = namespace
		temp["metric_name"] = metricName
		temp["from"] = strconv.Itoa(int(opts.StartTime.UnixMilli()))
		temp["to"] = strconv.Itoa(int(opts.EndTime.UnixMilli()))
		temp["period"] = "1"
		temp["filter"] = "average"
		temp["dim.0"] = fmt.Sprintf("%s,%s", dimesionName, opts.ResourceId)
		resp, err := self.commonMonitor(temp)
		if err != nil {
			log.Errorf("get monitor err:%s,input:%v", err.Error(), jsonutils.Marshal(temp))
			continue
		}
		metricData := MetricData{}
		err = resp.Unmarshal(&metricData)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		ret := cloudprovider.MetricValues{
			Id:     opts.ResourceId,
			Unit:   metricData.Unit,
			Values: []cloudprovider.MetricValue{},
		}
		tags := map[string]string{}
		switch metricData.MetricName {
		case "rds001_cpu_util":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_CPU_USAGE
		case "rds002_mem_util":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_MEM_USAGE
		case "rds004_bytes_in":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX
		case "rds005_bytes_out":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX
		case "rds039_disk_util":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_DISK_USAGE
		case "rds049_disk_read_throughput":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_DISK_READ_BPS
		case "rds050_disk_write_throughput":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_DISK_WRITE_BPS
		case "rds006_conn_count":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_CONN_COUNT
		case "rds008_qps":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_QPS
		case "rds009_tps":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_TPS
		case "rds013_innodb_reads":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_INNODB_READ_BPS
		case "rds014_innodb_writes":
			ret.MetricType = cloudprovider.RDS_METRIC_TYPE_INNODB_WRITE_BPS
		default:
			log.Warningf("invalid metricName %s for %s %s", metricData.MetricName, opts.ResourceType, opts.ResourceId)
			continue
		}
		for _, value := range metricData.Datapoints {
			metricValue := cloudprovider.MetricValue{
				Value:     value.Average,
				Timestamp: time.UnixMilli(value.Timestamp),
				Tags:      tags,
			}
			ret.Values = append(ret.Values, metricValue)
		}
		result = append(result, ret)
	}
	return result, nil
}

func (self *SHuaweiClient) getBucketMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	result := []cloudprovider.MetricValues{}
	namespace, dimesionName, metricNames := "SYS.OBS", "bucket_name", []string{
		"download_bytes",
		"upload_bytes",
		"first_byte_latency",
		"get_request_count",
		"request_count_4xx",
		"request_count_5xx",
	}

	for _, metricName := range metricNames {
		temp := make(map[string]string)
		temp["namespace"] = namespace
		temp["metric_name"] = metricName
		temp["from"] = strconv.Itoa(int(opts.StartTime.UnixMilli()))
		temp["to"] = strconv.Itoa(int(opts.EndTime.UnixMilli()))
		temp["period"] = "1"
		temp["filter"] = "average"
		temp["dim.0"] = fmt.Sprintf("%s,%s", dimesionName, opts.ResourceId)
		resp, err := self.commonMonitor(temp)
		if err != nil {
			log.Errorf("get monitor err:%s,input:%v", err.Error(), jsonutils.Marshal(temp))
			continue
		}
		metricData := MetricData{}
		err = resp.Unmarshal(&metricData)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		ret := cloudprovider.MetricValues{
			Id:     opts.ResourceId,
			Unit:   metricData.Unit,
			Values: []cloudprovider.MetricValue{},
		}
		tags := map[string]string{}
		switch metricData.MetricName {
		case "download_bytes":
			ret.MetricType = cloudprovider.BUCKET_METRIC_TYPE_NET_BPS_TX
		case "upload_bytes":
			ret.MetricType = cloudprovider.BUCKET_METRIC_TYPE_NET_BPS_RX
		case "first_byte_latency":
			ret.MetricType = cloudprovider.BUCKET_METRIC_TYPE_LATECY
			tags = map[string]string{"request": "get"}
		case "get_request_count":
			ret.MetricType = cloudprovider.BUCKET_METRYC_TYPE_REQ_COUNT
			tags = map[string]string{"request": "get"}
		case "request_count_4xx":
			ret.MetricType = cloudprovider.BUCKET_METRYC_TYPE_REQ_COUNT
			tags = map[string]string{"request": "4xx"}
		case "request_count_5xx":
			ret.MetricType = cloudprovider.BUCKET_METRYC_TYPE_REQ_COUNT
			tags = map[string]string{"request": "5xx"}
		default:
			log.Warningf("invalid metricName %s for %s %s", metricData.MetricName, opts.ResourceType, opts.ResourceId)
			continue
		}
		for _, value := range metricData.Datapoints {
			metricValue := cloudprovider.MetricValue{
				Value:     value.Average,
				Timestamp: time.UnixMilli(value.Timestamp),
				Tags:      tags,
			}
			ret.Values = append(ret.Values, metricValue)
		}
		result = append(result, ret)

	}
	return result, nil
}

func (self *SHuaweiClient) getLoadbalancerMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	namespace, dimesionName, metricNames := "SYS.ELB", "lbaas_instance_id", []string{
		"m1_cps",
		"m2_act_conn",
		"m7_in_Bps",
		"m8_out_Bps",
		"mc_l7_http_2xx",
		"md_l7_http_3xx",
		"me_l7_http_4xx",
		"mf_l7_http_5xx",
	}
	result := []cloudprovider.MetricValues{}

	for _, metricName := range metricNames {
		temp := make(map[string]string)
		temp["namespace"] = namespace
		temp["metric_name"] = metricName
		temp["from"] = strconv.Itoa(int(opts.StartTime.UnixMilli()))
		temp["to"] = strconv.Itoa(int(opts.EndTime.UnixMilli()))
		temp["period"] = "1"
		temp["filter"] = "average"
		temp["dim.0"] = fmt.Sprintf("%s,%s", dimesionName, opts.ResourceId)
		resp, err := self.commonMonitor(temp)
		if err != nil {
			return nil, err
		}
		metricData := MetricData{}
		err = resp.Unmarshal(&metricData)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		ret := cloudprovider.MetricValues{
			Id:     opts.ResourceId,
			Unit:   metricData.Unit,
			Values: []cloudprovider.MetricValue{},
		}
		tags := map[string]string{}
		switch metricData.MetricName {
		case "m1_cps":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_MAX_CONNECTION
		case "m2_act_conn":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_NET_ACTIVE_CONNECTION
		case "m7_in_Bps":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_NET_BPS_RX
		case "m8_out_Bps":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_NET_BPS_TX
		case "mc_l7_http_2xx":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_HRSP_COUNT
			tags = map[string]string{"request": "2xx"}
		case "md_l7_http_3xx":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_HRSP_COUNT
			tags = map[string]string{"request": "3xx"}
		case "md_l7_http_4xx":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_HRSP_COUNT
			tags = map[string]string{"request": "4xx"}
		case "md_l7_http_5xx":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_HRSP_COUNT
			tags = map[string]string{"request": "5xx"}
		case "me_l7_http_4xx":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_HRSP_COUNT
			tags = map[string]string{"request": "4xx"}
		case "mf_l7_http_5xx":
			ret.MetricType = cloudprovider.LB_METRIC_TYPE_HRSP_COUNT
			tags = map[string]string{"request": "5xx"}
		default:
			log.Warningf("invalid metricName %s for %s %s", metricData.MetricName, opts.ResourceType, opts.ResourceId)
			continue
		}
		for _, value := range metricData.Datapoints {
			metricValue := cloudprovider.MetricValue{
				Value:     value.Average,
				Timestamp: time.UnixMilli(value.Timestamp),
				Tags:      tags,
			}
			ret.Values = append(ret.Values, metricValue)
		}
		result = append(result, ret)
	}
	return result, nil
}

func (self *SHuaweiClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.getServerMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_RDS:
		return self.getRdsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_BUCKET:
		return self.getBucketMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_MODELARTS_POOL:
		return self.getModelartsPoolMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_LB:
		return self.getLoadbalancerMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.ResourceType)
	}
}
