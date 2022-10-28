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

package jdcloud

import (
	"fmt"
	"strconv"
	"time"

	jc_apis "github.com/jdcloud-api/jdcloud-sdk-go/services/monitor/apis"
	client "github.com/jdcloud-api/jdcloud-sdk-go/services/monitor/client"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SJDCloudClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_RDS:
		return self.GetRdsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SJDCloudClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricName := range map[cloudprovider.TMetricType]string{
		cloudprovider.VM_METRIC_TYPE_CPU_USAGE:          "cpu_util",
		cloudprovider.VM_METRIC_TYPE_MEM_USAGE:          "memory.usage",
		cloudprovider.VM_METRIC_TYPE_DISK_USAGE:         "vm.disk.dev.used",
		cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:         "vm.network.dev.bytes.in",
		cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:         "vm.network.dev.bytes.out",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:   "vm.disk.dev.bytes.read",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:  "vm.disk.dev.bytes.write",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:  "vm.disk.dev.io.read",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS: "vm.disk.dev.io.write",
	} {

		aggrType, downSampleType, serviceCode := "avg", "avg", "vm"
		startTime, endTime := opts.StartTime.Format(timeutils.FullIsoTimeFormat), opts.EndTime.Format(timeutils.FullIsoTimeFormat)
		request := jc_apis.NewDescribeMetricDataRequestWithAllParams(opts.RegionExtId, metricName, &aggrType, &downSampleType, &startTime, &endTime, nil, nil, nil, nil, &serviceCode, nil, opts.ResourceId)
		monitorClient := client.NewMonitorClient(self.getCredential())
		monitorClient.Logger = Logger{debug: self.debug}
		response, err := monitorClient.DescribeMetricData(request)
		if err != nil {
			return nil, err
		}
		metric := cloudprovider.MetricValues{}
		metric.Id = opts.ResourceId
		metric.MetricType = metricType
		for _, data := range response.Result.MetricDatas {
			for _, value := range data.Data {
				metricValue := cloudprovider.MetricValue{}
				metricValue.Timestamp = time.Unix(value.Timestamp/1000, 0)
				if value.Value == nil {
					continue
				}
				metricValue.Value, err = strconv.ParseFloat(fmt.Sprintf("%s", value.Value), 64)
				if err != nil {
					log.Errorf("parse value %v error: %v", value.Value, nil)
					continue
				}
				metric.Values = append(metric.Values, metricValue)
			}
		}
		ret = append(ret, metric)
	}
	return ret, nil
}

func (self *SJDCloudClient) GetRdsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics := map[cloudprovider.TMetricType]string{}
	switch opts.Engine {
	case api.DBINSTANCE_TYPE_MYSQL, api.DBINSTANCE_TYPE_SQLSERVER, api.DBINSTANCE_TYPE_PERCONA, api.DBINSTANCE_TYPE_MARIADB:
		metrics = map[cloudprovider.TMetricType]string{
			cloudprovider.RDS_METRIC_TYPE_CPU_USAGE:  "database.docker.cpu.util",
			cloudprovider.RDS_METRIC_TYPE_MEM_USAGE:  "database.docker.memory.pused",
			cloudprovider.RDS_METRIC_TYPE_DISK_USAGE: "database.docker.disk1.used",
			cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX: "database.docker.network.incoming",
			cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX: "database.docker.network.outgoing",
		}
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		metrics = map[cloudprovider.TMetricType]string{
			cloudprovider.RDS_METRIC_TYPE_CPU_USAGE:  "database.docker.cpu.util",
			cloudprovider.RDS_METRIC_TYPE_MEM_USAGE:  "database.docker.memory.pused",
			cloudprovider.RDS_METRIC_TYPE_DISK_USAGE: "database.docker.disk1.used",
			cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX: "database.docker.network.incoming",
			cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX: "database.docker.network.outgoing",
		}
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.Engine)
	}
	ret := []cloudprovider.MetricValues{}
	aggrType, downSampleType := "avg", "avg"
	startTime, endTime := opts.StartTime.Format(timeutils.FullIsoTimeFormat), opts.EndTime.Format(timeutils.FullIsoTimeFormat)
	serviceCode := map[string]string{
		api.DBINSTANCE_TYPE_SQLSERVER:  "sqlserver",
		api.DBINSTANCE_TYPE_MYSQL:      "database",
		api.DBINSTANCE_TYPE_PERCONA:    "percona",
		api.DBINSTANCE_TYPE_MARIADB:    "mariadb",
		api.DBINSTANCE_TYPE_POSTGRESQL: "pag",
	}[opts.Engine]
	for metricType, metricName := range metrics {
		request := jc_apis.NewDescribeMetricDataRequestWithAllParams(opts.RegionExtId, metricName, &aggrType, &downSampleType, &startTime, &endTime, nil, nil, nil, nil, &serviceCode, nil, opts.ResourceId)
		monitorClient := client.NewMonitorClient(self.getCredential())
		monitorClient.Logger = Logger{}
		response, err := monitorClient.DescribeMetricData(request)
		if err != nil {
			return nil, err
		}
		metric := cloudprovider.MetricValues{}
		metric.Id = opts.ResourceId
		metric.MetricType = metricType
		for _, data := range response.Result.MetricDatas {
			for _, value := range data.Data {
				metricValue := cloudprovider.MetricValue{}
				metricValue.Timestamp = time.Unix(value.Timestamp/1000, 0)
				if value.Value == nil {
					continue
				}
				metricValue.Value, err = strconv.ParseFloat(fmt.Sprintf("%s", value.Value), 64)
				if err != nil {
					log.Errorf("parse value %v error: %v", value.Value, nil)
					continue
				}
				// kbps -> byte
				if opts.Engine != api.DBINSTANCE_TYPE_POSTGRESQL && (metricType == cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX || metricType == cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX) {
					metricValue.Value = metricValue.Value / 1024.0
				}
				metric.Values = append(metric.Values, metricValue)
			}
		}
		ret = append(ret, metric)
	}
	return ret, nil
}
