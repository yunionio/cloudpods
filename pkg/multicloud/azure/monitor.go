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

package azure

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type ResponseMetirc struct {
	// Cost - The integer value representing the cost of the query, for data case.
	Cost float64 `json:"cost,omitempty"`
	// Timespan - The timespan for which the data was retrieved. Its value consists of two datetimes concatenated, separated by '/'.  This may be adjusted in the future and returned back from what was originally requested.
	Timespan string `json:"timespan,omitempty"`
	// Interval - The interval (window size) for which the metric data was returned in.  This may be adjusted in the future and returned back from what was originally requested.  This is not present if a metadata request was made.
	Interval string `json:"interval,omitempty"`
	// Namespace - The namespace of the metrics been queried
	Namespace string `json:"namespace,omitempty"`
	// Resourceregion - The region of the resource been queried for metrics.
	Resourceregion string `json:"resourceregion,omitempty"`
	// Value - the value of the collection.
	Value []Metric `json:"value,omitempty"`
}

// Metric the result data of a query.
type Metric struct {
	// ID - the metric Id.
	ID string `json:"id,omitempty"`
	// Type - the resource type of the metric resource.
	Type string `json:"type,omitempty"`
	// Name - the name and the display name of the metric, i.e. it is localizable string.
	Name LocalizableString `json:"name,omitempty"`
	// Unit - the unit of the metric. Possible values include: 'UnitCount', 'UnitBytes', 'UnitSeconds', 'UnitCountPerSecond', 'UnitBytesPerSecond', 'UnitPercent', 'UnitMilliSeconds', 'UnitByteSeconds', 'UnitUnspecified', 'UnitCores', 'UnitMilliCores', 'UnitNanoCores', 'UnitBitsPerSecond'
	Unit string `json:"unit,omitempty"`
	// Timeseries - the time series returned when a data query is performed.
	Timeseries []TimeSeriesElement `json:"timeseries,omitempty"`
}

type TimeSeriesElement struct {
	// Metadatavalues - the metadata values returned if $filter was specified in the call.
	Metadatavalues []MetadataValue `json:"metadatavalues,omitempty"`
	// Data - An array of data points representing the metric values.  This is only returned if a result type of data is specified.
	Data []MetricValue `json:"data,omitempty"`
}

type MetadataValue struct {
	// Name - the name of the metadata.
	Name LocalizableString `json:"name,omitempty"`
	// Value - the value of the metadata.
	Value string `json:"value,omitempty"`
}

type LocalizableString struct {
	// Value - the invariant value.
	Value string `json:"value,omitempty"`
	// LocalizedValue - the locale specific value.
	LocalizedValue string `json:"localizedValue,omitempty"`
}

type MetricValue struct {
	// TimeStamp - the timestamp for the metric value in ISO 8601 format.
	TimeStamp time.Time `json:"timeStamp,omitempty"`
	// Average - the average value in the time range.
	Average float64 `json:"average,omitempty"`
	// Minimum - the least value in the time range.
	Minimum float64 `json:"minimum,omitempty"`
	// Maximum - the greatest value in the time range.
	Maximum float64 `json:"maximum,omitempty"`
	// Total - the sum of all of the values in the time range.
	Total float64 `json:"total,omitempty"`
	// Count - the number of samples in the time range. Can be used to determine the number of values that contributed to the average value.
	Count float64 `json:"count,omitempty"`
}

func (self MetricValue) GetValue() float64 {
	return self.Average + self.Minimum + self.Maximum + self.Total + self.Count
}

func (self *SRegion) GetMonitorData(name string, ns string, resourceId string, since time.Time, until time.Time, interval string, filter string) (*ResponseMetirc, error) {
	params := url.Values{}
	params.Set("metricnamespace", ns)
	params.Set("metricnames", name)
	params.Set("interval", "PT1M")
	if len(interval) != 0 {
		params.Set("interval", interval)
	}
	params.Set("aggregation", "Average")
	params.Set("api-version", "2018-01-01")
	if !since.IsZero() && !until.IsZero() {
		params.Set("timespan", since.UTC().Format(time.RFC3339)+"/"+until.UTC().Format(time.RFC3339))
	}
	if len(filter) != 0 {
		params.Set("$filter", filter)
	}
	resource := fmt.Sprintf("%s/providers/microsoft.insights/metrics", resourceId)
	elements := ResponseMetirc{}
	err := self.get(resource, params, &elements)
	if err != nil {
		return nil, err
	}
	return &elements, nil
}

const (
	RDS_TYPE_SERVERS          = "servers"
	RDS_TYPE_FLEXIBLE_SERVERS = "flexibleServers"
)

// https://docs.microsoft.com/en-us/azure/azure-monitor/essentials/metrics-supported
func (self *SAzureClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	metricnamespace, metricnames := "", ""
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_RDS:
		rdsType := RDS_TYPE_SERVERS
		if strings.Contains(opts.ResourceId, strings.ToLower(RDS_TYPE_FLEXIBLE_SERVERS)) {
			rdsType = RDS_TYPE_FLEXIBLE_SERVERS
		}
		switch opts.Engine {
		case api.DBINSTANCE_TYPE_MYSQL:
			metricnamespace = fmt.Sprintf("Microsoft.DBforMySQL/%s", rdsType)
			metricnames = "cpu_percent,memory_percent,storage_percent,network_bytes_ingress,network_bytes_egress,io_consumption_percent,connections_failed,active_connections"
		case api.DBINSTANCE_TYPE_MARIADB:
			metricnamespace = "Microsoft.DBforMariaDB/servers"
			metricnames = "cpu_percent,memory_percent,storage_percent,network_bytes_ingress,network_bytes_egress,io_consumption_percent,connections_failed,active_connections"
		case api.DBINSTANCE_TYPE_SQLSERVER:
			metricnamespace = "Microsoft.Sql/servers/databases"
			metricnames = "cpu_percent,connection_successful,sqlserver_process_memory_percent,connection_successful,connection_failed"
			result := struct {
				Value []SSQLServerDatabase
			}{}
			err := self.get(opts.ResourceId+"/databases", url.Values{}, &result)
			if err != nil {
				return nil, err
			}
			for i := range result.Value {
				if result.Value[i].Name == "master" {
					continue
				}
				metrics, err := self.getMetricValues(result.Value[i].ID, metricnamespace, metricnames, result.Value[i].Name, opts.StartTime, opts.EndTime)
				if err != nil {
					log.Errorf("error: %v", err)
					continue
				}
				for j := range metrics {
					metrics[j].Id = opts.ResourceId
					ret = append(ret, metrics[j])
				}
			}
			return ret, nil
		case api.DBINSTANCE_TYPE_POSTGRESQL:
			metricnamespace = fmt.Sprintf("Microsoft.DBforPostgreSQL/%s", rdsType)
			metricnames = "cpu_percent,memory_percent,storage_percent,network_bytes_ingress,network_bytes_egress,io_consumption_percent,connections_failed,active_connections"
		default:
			return nil, errors.Wrapf(cloudprovider.ErrNotSupported, opts.Engine)
		}
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		metricnamespace = "Microsoft.Compute/virtualMachines"
		metricnames = "Percentage CPU,Network In Total,Network Out Total,Disk Read Bytes,Disk Write Bytes,Disk Read Operations/Sec,Disk Write Operations/Sec"
		if strings.Contains(opts.ResourceId, "microsoft.classiccompute/virtualmachines") {
			metricnamespace = "microsoft.classiccompute/virtualmachines"
		}
	case cloudprovider.METRIC_RESOURCE_TYPE_REDIS:
		metricnamespace = "Microsoft.Cache/redis"
		metricnames = "percentProcessorTime,usedmemorypercentage,connectedclients,operationsPerSecond,alltotalkeys,expiredkeys,usedmemory,serverLoad,errors"
	case cloudprovider.METRIC_RESOURCE_TYPE_LB:
		metricnamespace = "Microsoft.Network/loadBalancers"
		metricnames = "SnatConnectionCount,UsedSnatPorts"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.ResourceType)
	}
	return self.getMetricValues(opts.ResourceId, metricnamespace, metricnames, "", opts.StartTime, opts.EndTime)
}

func (self *SAzureClient) getMetricValues(resourceId, metricnamespace, metricnames string, database string, startTime, endTime time.Time) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	params := url.Values{}
	params.Set("interval", "PT1M")
	params.Set("api-version", "2018-01-01")
	params.Set("timespan", startTime.UTC().Format(time.RFC3339)+"/"+endTime.UTC().Format(time.RFC3339))
	resource := fmt.Sprintf("%s/providers/microsoft.insights/metrics", resourceId)
	params.Set("aggregation", "Average,Count,Maximum,Total")
	params.Set("metricnamespace", metricnamespace)
	params.Set("metricnames", metricnames)
	elements := ResponseMetirc{}
	err := self.get(resource, params, &elements)
	if err != nil {
		return ret, err
	}
	for _, element := range elements.Value {
		metric := cloudprovider.MetricValues{
			Unit: element.Unit,
		}
		switch element.Name.Value {
		case "cpu_percent":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_CPU_USAGE
		case "memory_percent", "sqlserver_process_memory_percent":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_MEM_USAGE
		case "storage_percent":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_DISK_USAGE
		case "network_bytes_ingress":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX
		case "network_bytes_egress":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX
		case "io_consumption_percent":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_DISK_IO_PERCENT
		case "connections_failed", "connection_failed":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_CONN_FAILED
		case "active_connections", "connection_successful":
			metric.MetricType = cloudprovider.RDS_METRIC_TYPE_CONN_ACTIVE
		case "Percentage CPU":
			metric.MetricType = cloudprovider.VM_METRIC_TYPE_CPU_USAGE
		case "Network In Total":
			metric.MetricType = cloudprovider.VM_METRIC_TYPE_NET_BPS_RX
		case "Network Out Total":
			metric.MetricType = cloudprovider.VM_METRIC_TYPE_NET_BPS_TX
		case "Disk Read Bytes":
			metric.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS
		case "Disk Write Bytes":
			metric.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS
		case "Disk Read Operations/Sec":
			metric.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS
		case "Disk Write Operations/Sec":
			metric.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS
		case "percentProcessorTime":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_CPU_USAGE
		case "usedmemorypercentage":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_MEM_USAGE
		case "connectedclients":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_CONN_USAGE
		case "operationsPerSecond":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_OPT_SES
		case "alltotalkeys":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_CACHE_KEYS
		case "expiredkeys":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_CACHE_EXP_KEYS
		case "usedmemory":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_DATA_MEM_USAGE
		case "serverLoad":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_SERVER_LOAD
		case "errors":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_CONN_ERRORS
		case "SnatConnectionCount":
			metric.MetricType = cloudprovider.LB_METRIC_TYPE_SNAT_PORT
		case "UsedSnatPorts":
			metric.MetricType = cloudprovider.LB_METRIC_TYPE_SNAT_CONN_COUNT
		default:
			log.Warningf("incognizance metric type %s", element.Name.Value)
			continue
		}
		for _, timeserie := range element.Timeseries {
			for _, data := range timeserie.Data {
				tags := map[string]string{}
				if len(database) > 0 {
					tags["database"] = database
				}
				metric.Values = append(metric.Values, cloudprovider.MetricValue{
					Timestamp: data.TimeStamp,
					Value:     data.GetValue(),
					Tags:      tags,
				})
			}
		}
		ret = append(ret, metric)
	}
	return ret, nil
}
