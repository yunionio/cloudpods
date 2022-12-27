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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
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
	if self.Average > 0 {
		return self.Average
	}
	if self.Total > 0 {
		return self.Total
	}
	if self.Count > 0 {
		return self.Count
	}
	if self.Minimum > 0 {
		return self.Minimum
	}
	return self.Maximum
}

const (
	RDS_TYPE_SERVERS          = "servers"
	RDS_TYPE_FLEXIBLE_SERVERS = "flexibleServers"
)

// https://docs.microsoft.com/en-us/azure/azure-monitor/essentials/metrics-supported
func (self *SAzureClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_RDS:
		return self.GetRdsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_REDIS:
		return self.GetRedisMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_LB:
		return self.GetLbMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_K8S:
		return self.GetK8sMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.ResourceType)
	}
}

type AzureTableMetricDataValue struct {
	Timestamp    time.Time
	Average      float64
	Count        float64
	CounterName  string
	DeploymentId string
	Host         string
	Total        float64
	Maximum      float64
	Minimum      float64
}

func (self AzureTableMetricDataValue) GetValue() float64 {
	if self.Average > 0 {
		return self.Average
	}
	if self.Count > 0 {
		return self.Count
	}
	if self.Minimum > 0 {
		return self.Minimum
	}
	return self.Maximum
}

type AzureTableMetricData struct {
	Value []AzureTableMetricDataValue
}

func (self *SAzureClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricnamespace := "Microsoft.Compute/virtualMachines"
	metricnames := "Percentage CPU,Network In Total,Network Out Total,Disk Read Bytes,Disk Write Bytes,Disk Read Operations/Sec,Disk Write Operations/Sec"
	if strings.Contains(opts.ResourceId, "microsoft.classiccompute/virtualmachines") {
		metricnamespace = "microsoft.classiccompute/virtualmachines"
	}
	ret, err := self.getMetricValues(opts.ResourceId, metricnamespace, metricnames, nil, "", opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, err
	}

	for metricType, names := range map[cloudprovider.TMetricType][]string{
		cloudprovider.VM_METRIC_TYPE_MEM_USAGE:  {"/builtin/memory/percentusedmemory", "\\Memory\\% Committed Bytes In Use"},
		cloudprovider.VM_METRIC_TYPE_DISK_USAGE: {"/builtin/filesystem/percentusedspace", "\\LogicalDisk(_Total)\\% Free Space"},
	} {
		filters := []string{}
		for _, name := range names {
			filter := fmt.Sprintf("name.value eq '%s'", name)
			filters = append(filters, filter)
		}
		metricDefinitions, err := self.getMetricDefinitions(opts.ResourceId, strings.Join(filters, " or "))
		if err != nil {
			log.Errorf("getMetricDefinitions error: %v", err)
			continue
		}
		metric := cloudprovider.MetricValues{}
		metric.MetricType = metricType
		header := http.Header{}
		header.Set("accept", "application/json;odata=minimalmetadata")
		for _, definition := range metricDefinitions.Value {
			for _, tables := range definition.MetricAvailabilities {
				if tables.TimeGrain != "PT1M" {
					continue
				}
				for _, table := range tables.Location.TableInfo {
					if table.EndTime.Before(opts.EndTime) {
						continue
					}
					url := fmt.Sprintf("%s%s%s", tables.Location.TableEndpoint, table.TableName, table.SasToken)
					_, resp, err := httputils.JSONRequest(httputils.GetDefaultClient(), context.Background(), httputils.GET, url, header, nil, self.debug)
					if err != nil {
						log.Errorf("request %s error: %v", url, err)
						continue
					}
					values := &AzureTableMetricData{}
					resp.Unmarshal(values)
					for _, v := range values.Value {
						name := strings.ReplaceAll(v.CounterName, `\\`, `\`)
						if !utils.IsInStringArray(name, names) {
							continue
						}
						if v.Timestamp.After(opts.StartTime) && v.Timestamp.Before(opts.EndTime) {
							value := v.GetValue()
							if metricType == cloudprovider.VM_METRIC_TYPE_DISK_USAGE && strings.Contains(strings.ToLower(name), "free") {
								value = 100 - value
							}
							metric.Values = append(metric.Values, cloudprovider.MetricValue{
								Timestamp: v.Timestamp,
								Value:     value,
							})
						}
					}
				}
			}
		}
		if len(metric.Values) > 0 {
			ret = append(ret, metric)
		}
	}

	if len(opts.OsType) == 0 || strings.Contains(strings.ToLower(opts.OsType), "win") {
		workspaces, err := self.GetLoganalyticsWorkspaces()
		if err != nil {
			return ret, nil
		}
		winmetric := cloudprovider.MetricValues{}
		winmetric.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_USAGE
		winmetric.Values = []cloudprovider.MetricValue{}
		for i := range workspaces {
			data, err := self.GetInstanceDiskUsage(workspaces[i].SLoganalyticsWorkspaceProperties.CustomerId, opts.ResourceId, opts.StartTime, opts.EndTime)
			if err != nil {
				continue
			}
			for i := range data {
				for j := range data[i].Rows {
					if len(data[i].Rows[j]) == 5 {
						date, device, free := data[i].Rows[j][0], data[i].Rows[j][1], data[i].Rows[j][2]
						dataTime, err := timeutils.ParseTimeStr(date)
						if err != nil {
							continue
						}
						if dataTime.Second() > 15 {
							continue
						}
						freeSize, err := strconv.ParseFloat(free, 64)
						if err != nil {
							continue
						}
						winmetric.Values = append(winmetric.Values, cloudprovider.MetricValue{
							Timestamp: dataTime,
							Value:     100 - freeSize,
							Tags:      map[string]string{cloudprovider.METRIC_TAG_DEVICE: device},
						})
					}
				}
			}
		}
		if len(winmetric.Values) > 0 {
			ret = append(ret, winmetric)
		}
	}

	return ret, nil
}

func (self *SAzureClient) GetRedisMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricnamespace := "Microsoft.Cache/redis"
	metricnames := "percentProcessorTime,usedmemorypercentage,connectedclients,operationsPerSecond,alltotalkeys,expiredkeys,usedmemory"
	return self.getMetricValues(opts.ResourceId, metricnamespace, metricnames, nil, "", opts.StartTime, opts.EndTime)
}

func (self *SAzureClient) GetLbMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricnamespace := "Microsoft.Network/loadBalancers"
	metricnames := "SnatConnectionCount,UsedSnatPorts"
	return self.getMetricValues(opts.ResourceId, metricnamespace, metricnames, nil, "", opts.StartTime, opts.EndTime)
}

func (self *SAzureClient) GetK8sMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricnamespace := "Microsoft.ContainerService/managedClusters"
	metricnames := "node_cpu_usage_percentage,node_memory_rss_percentage,node_disk_usage_percentage,node_network_in_bytes,node_network_out_bytes"
	filter := fmt.Sprintf("node eq '*'")
	return self.getMetricValues(opts.ResourceId, metricnamespace, metricnames, nil, filter, opts.StartTime, opts.EndTime)
}

func (self *SAzureClient) GetRdsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	metricnamespace, metricnames := "", ""
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
			metrics, err := self.getMetricValues(result.Value[i].ID, metricnamespace, metricnames, map[string]string{cloudprovider.METRIC_TAG_DATABASE: result.Value[i].Name}, "", opts.StartTime, opts.EndTime)
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
	return self.getMetricValues(opts.ResourceId, metricnamespace, metricnames, nil, "", opts.StartTime, opts.EndTime)
}

type MetrifDefinitions struct {
	Id    string
	Value []MetrifDefinition
}

type MetrifDefinition struct {
	Name struct {
		Value          string
		LocalizedValue string
	}
	Category               string
	Unit                   string
	PrimaryAggregationType string
	ResourceUri            string
	ResourceId             string
	MetricAvailabilities   []struct {
		TimeGrain string
		Retention string
		Location  struct {
			TableEndpoint string
			TableInfo     []struct {
				TableName              string
				StartTime              time.Time
				EndTime                time.Time
				SasToken               string
				SasTokenExpirationTime string
			}
			PartitionKey string
		}
	}
	Id string
}

func (self *SAzureClient) getMetricDefinitions(resourceId, filter string) (*MetrifDefinitions, error) {
	params := url.Values{}
	params.Set("api-version", "2015-07-01")
	resource := fmt.Sprintf("%s/providers/microsoft.insights/metricDefinitions", resourceId)
	if len(filter) > 0 {
		params.Set("$filter", filter)
	}
	result := &MetrifDefinitions{}
	err := self.get(resource, params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (self *SAzureClient) getMetricValues(resourceId, metricnamespace, metricnames string, metricTag map[string]string, filter string, startTime, endTime time.Time) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	params := url.Values{}
	params.Set("interval", "PT1M")
	params.Set("api-version", "2018-01-01")
	params.Set("timespan", startTime.UTC().Format(time.RFC3339)+"/"+endTime.UTC().Format(time.RFC3339))
	resource := fmt.Sprintf("%s/providers/microsoft.insights/metrics", resourceId)
	params.Set("aggregation", "Average,Count,Maximum,Total")
	params.Set("metricnamespace", metricnamespace)
	params.Set("metricnames", metricnames)
	if len(filter) > 0 {
		params.Set("$filter", filter)
	}
	elements := ResponseMetirc{}
	err := self.get(resource, params, &elements)
	if err != nil {
		return ret, err
	}
	for i := range elements.Value {
		element := elements.Value[i]
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
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_USED_CONN
		case "operationsPerSecond":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_OPT_SES
		case "alltotalkeys":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_CACHE_KEYS
		case "expiredkeys":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_CACHE_EXP_KEYS
		case "usedmemory":
			metric.MetricType = cloudprovider.REDIS_METRIC_TYPE_DATA_MEM_USAGE
		case "SnatConnectionCount":
			metric.MetricType = cloudprovider.LB_METRIC_TYPE_SNAT_PORT
		case "UsedSnatPorts":
			metric.MetricType = cloudprovider.LB_METRIC_TYPE_SNAT_CONN_COUNT
		case "node_cpu_usage_percentage":
			metric.MetricType = cloudprovider.K8S_NODE_METRIC_TYPE_CPU_USAGE
		case "node_memory_rss_percentage":
			metric.MetricType = cloudprovider.K8S_NODE_METRIC_TYPE_MEM_USAGE
		case "node_disk_usage_percentage":
			metric.MetricType = cloudprovider.K8S_NODE_METRIC_TYPE_DISK_USAGE
		case "node_network_in_bytes":
			metric.MetricType = cloudprovider.K8S_NODE_METRIC_TYPE_NET_BPS_RX
		case "node_network_out_bytes":
			metric.MetricType = cloudprovider.K8S_NODE_METRIC_TYPE_NET_BPS_TX
		default:
			log.Warningf("incognizance metric type %s", element.Name.Value)
			continue
		}
		for _, timeserie := range element.Timeseries {
			tags := map[string]string{}
			for _, metadata := range timeserie.Metadatavalues {
				if metadata.Name.Value == "node" { //k8s node
					tags[cloudprovider.METRIC_TAG_NODE] = metadata.Value
				}
			}
			for k, v := range metricTag {
				tags[k] = v
			}
			for _, data := range timeserie.Data {
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
