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

package common

import (
	"context"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	o "yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type MonType string

const (
	SERVER       MonType = "server"
	HOST         MonType = "host"
	REDIS        MonType = "redis"
	RDS          MonType = "rds"
	OSS          MonType = "oss"
	ELB          MonType = "elb"
	CLOUDACCOUNT MonType = "cloudaccount"
	STORAGE      MonType = "storage"
	ALERT_RECORD MonType = "alertRecord"
)

const (
	KEY_LIMIT  = "limit"
	KEY_ADMIN  = "admin"
	KEY_USABLE = "usable"
	DETAILS    = "details"
	KEY_SCOPE  = "scope"
)

const (
	TYPE_VIRTUALMACHINE = "VirtualMachine"
	TYPE_HOSTSYSTEM     = "HostSystem"
)

var OtherVmTags = map[string]string{
	"source":   "cloudmon",
	"res_type": "guest",
	"is_vm":    "true",
}

var OtherHostTag = map[string]string{
	"source":   "cloudmon",
	"res_type": "host",
	"is_vm":    "false",
}

type ReportOptions struct {
	Batch     int      `help:"batch"`
	Count     int      `help:"count" json:"count"`
	Interval  string   `help:"interval""`
	Timeout   int64    `help:"command timeout unit:second" default:"10"`
	SinceTime string   `help:"sinceTime"`
	EndTime   string   `help:"endTime"`
	Provider  []string `help:"List objects from the provider" choices:"VMware|Aliyun|Qcloud|Azure|Aws|Huawei|ZStack|Google|Apsara|JDcloud|Ecloud" json:"provider,omitempty"`
	PingProbeOptions
}

type PingProbeOptions struct {
	Debug         bool `help:"debug"`
	ProbeCount    int  `help:"probe count, default is 3" default:"3"`
	TimeoutSecond int  `help:"probe timeout in second, default is 1 second" default:"1"`
}

var InstanceProviders = "Aliyun,Azure,Aws,Qcloud,VMWare,Huawei,Openstack,Ucloud,ZStack"

//server的key-value对应保存时的Tags和Pairs
//var ServerTags = []string{"host", "host_id", "vm_id", "vm_ip", "vm_name", "zone", "zone_id", "zone_ext_id",
//	"hypervisor", "os_type", "status", "region", "region_id", "region_ext_id", "tenant", "tenant_id", "brand", "name"}
var ServerTags = map[string]string{
	"host":             "host",
	"host_id":          "host_id",
	"id":               "vm_id",
	"ips":              "vm_ip",
	"name":             "vm_name",
	"zone":             "zone",
	"zone_id":          "zone_id",
	"zone_ext_id":      "zone_ext_id",
	"os_type":          "os_type",
	"status":           "status",
	"cloudregion":      "cloudregion",
	"cloudregion_id":   "cloudregion_id",
	"region_ext_id":    "region_ext_id",
	"tenant":           "tenant",
	"tenant_id":        "tenant_id",
	"brand":            "brand",
	"scaling_group_id": "vm_scaling_group_id",
	"domain_id":        "domain_id",
	"project_domain":   "project_domain",
}
var HostTags = map[string]string{
	"id":             "host_id",
	"ips":            "host_ip",
	"name":           "host",
	"zone":           "zone",
	"zone_id":        "zone_id",
	"zone_ext_id":    "zone_ext_id",
	"os_type":        "os_type",
	"status":         "status",
	"cloudregion":    "cloudregion",
	"cloudregion_id": "cloudregion_id",
	"region_ext_id":  "region_ext_id",
	"tenant":         "tenant",
	"tenant_id":      "tenant_id",
	"brand":          "brand",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
}
var RdsTags = map[string]string{
	"host":           "host",
	"host_id":        "host_id",
	"id":             "rds_id",
	"ips":            "rds_ip",
	"name":           "rds_name",
	"zone":           "zone",
	"zone_id":        "zone_id",
	"zone_ext_id":    "zone_ext_id",
	"os_type":        "os_type",
	"status":         "status",
	"cloudregion":    "cloudregion",
	"cloudregion_id": "cloudregion_id",
	"region_ext_id":  "region_ext_id",
	"tenant":         "tenant",
	"tenant_id":      "tenant_id",
	"brand":          "brand",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
}
var RedisTags = map[string]string{
	"host":           "host",
	"host_id":        "host_id",
	"id":             "redis_id",
	"ips":            "redis_ip",
	"name":           "redis_name",
	"zone":           "zone",
	"zone_id":        "zone_id",
	"zone_ext_id":    "zone_ext_id",
	"os_type":        "os_type",
	"status":         "status",
	"cloudregion":    "cloudregion",
	"cloudregion_id": "cloudregion_id",
	"region_ext_id":  "region_ext_id",
	"tenant":         "tenant",
	"tenant_id":      "tenant_id",
	"brand":          "brand",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
}
var OssTags = map[string]string{
	"host":           "host",
	"host_id":        "host_id",
	"id":             "oss_id",
	"ips":            "oss_ip",
	"name":           "oss_name",
	"zone":           "zone",
	"zone_id":        "zone_id",
	"zone_ext_id":    "zone_ext_id",
	"os_type":        "os_type",
	"status":         "status",
	"cloudregion":    "cloudregion",
	"cloudregion_id": "cloudregion_id",
	"region_ext_id":  "region_ext_id",
	"tenant":         "tenant",
	"tenant_id":      "tenant_id",
	"brand":          "brand",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
}
var ElbTags = map[string]string{
	"host":           "host",
	"host_id":        "host_id",
	"id":             "elb_id",
	"ips":            "elb_ip",
	"name":           "elb_name",
	"zone":           "zone",
	"zone_id":        "zone_id",
	"zone_ext_id":    "zone_ext_id",
	"os_type":        "os_type",
	"status":         "status",
	"region":         "region",
	"cloudregion":    "cloudregion",
	"cloudregion_id": "cloudregion_id",
	"tenant":         "tenant",
	"tenant_id":      "tenant_id",
	"brand":          "brand",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
}

var CloudAccountTags = map[string]string{
	"id":             "cloudaccount_id",
	"name":           "cloudaccount_name",
	"brand":          "brand",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
}

var StorageTags = map[string]string{
	"id":             "storage_id",
	"name":           "storage_name",
	"zone":           "zone",
	"zone_id":        "zone_id",
	"zone_ext_id":    "zone_ext_id",
	"status":         "status",
	"cloudregion":    "cloudregion",
	"cloudregion_id": "cloudregion_id",
	"region_ext_id":  "region_ext_id",
	"brand":          "brand",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
}

var AlertRecordHistoryTags = map[string]string{
	"alert_name":     "alert_name",
	"alert_id":       "alert_id",
	"domain_id":      "domain_id",
	"project_domain": "project_domain",
	"tenant_id":      "tenant_id",
	"tenant":         "tenant",
	"res_type":       "res_type",
}

var CloudAccountFields = []string{"balance"}

var AlertRecordHistoryFields = []string{"res_num"}

var ServerPairs = []string{"vcpu_count", "vmem_size", "disk"}

var AddTags = map[string]string{
	"source": "cloudmon",
}

//get substring from str before separator
func SubstringBefore(str, separator string) string {
	if str != "" {
		if separator == "" {
			return ""
		} else {
			if pos := strings.Index(str, separator); pos == -1 {
				return str
			} else {
				return str[0:pos]
			}
		}
	} else {
		return str
	}
}

//get substring from str after separator
func SubstringAfter(str, separator string) string {
	if str != "" {
		if separator == "" {
			return ""
		} else {
			if pos := strings.Index(str, separator); pos == -1 {
				return ""
			} else {
				return str[pos+len(separator):]
			}
		}
	} else {
		return str
	}
}

//get a substring from str between[open,close)
func SubstringBetween(str, open, close string) string {
	if str != "" && open != "" && close != "" {
		if start := strings.Index(str, open); start != -1 {
			if end := strings.Index(str[start+len(open):], close); end != -1 {
				return str[start+len(open) : start+len(open)+end]
			}
		}
		return ""
	} else {
		return ""
	}
}

func ParseTimeStr(startTime, endTime string) (since, util time.Time, err error) {
	since, err = timeutils.ParseTimeStr(startTime)
	if err != nil {
		return since, util, err
	}
	util, err = timeutils.ParseTimeStr(endTime)
	if err != nil {
		return since, util, err
	}
	return since, util, nil
}

func TimeRangeFromArgs(args *ReportOptions) (since, until time.Time, err error) {
	if args.SinceTime != "" && args.EndTime != "" {
		since, until, err = ParseTimeStr(args.SinceTime, args.EndTime)
		if err != nil {
			return since, until, err
		}
	} else {
		period64, err := strconv.ParseInt(args.Interval, 10, 8)
		if err != nil {
			return since, until, err
		}
		since = time.Now().Add(-time.Minute * time.Duration(period64))
		until = time.Now()
	}
	return since, until, nil
}

//组装server相关capability
func FillVMCapacity(server *jsonutils.JSONDict) (influxdb.SMetricData, error) {
	metric, err := JsonToMetric(server, "vm_capacity", ServerTags, ServerPairs)
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	hypevisor, _ := server.GetString("hypervisor")
	metric.Timestamp = time.Now()
	metric.Tags = append(metric.Tags, influxdb.SKeyValue{
		Key:   "res_type",
		Value: "guest",
	}, influxdb.SKeyValue{
		Key:   "is_vm",
		Value: "true",
	}, influxdb.SKeyValue{
		Key:   "platform",
		Value: hypevisor,
	})
	return metric, nil
}

func GetMeasurement(action string, influxDbSpec string) (measurement string) {
	// VirtualMachine -> VMware类型的虚拟机
	if action == TYPE_VIRTUALMACHINE {
		measurement = SubstringBefore(influxDbSpec, ".")
	}
	if action == TYPE_HOSTSYSTEM {
		measurement = SubstringBetween(influxDbSpec, "vm_", ".")
		if strings.Contains(influxDbSpec, "vm_netio") {
			measurement = "net"
		}
	}
	return measurement
}

func JsonToMetric(obj *jsonutils.JSONDict, name string, tags map[string]string, metrics []string) (influxdb.SMetricData, error) {
	metric := influxdb.SMetricData{Name: name}
	objMap, err := obj.GetMap()
	if err != nil {
		return metric, errors.Wrap(err, "obj.GetMap")
	}
	tagPairs := make([]influxdb.SKeyValue, 0)
	metricPairs := make([]influxdb.SKeyValue, 0)
	for k, v := range objMap {
		val, _ := v.GetString()
		if strings.Contains(k, "ip") {
			if strings.Contains(val, ",") {
				val = strings.ReplaceAll(val, ",", "|")
			}
		}
		if tag, ok := tags[k]; ok {
			tagPairs = append(tagPairs, influxdb.SKeyValue{
				Key:   tag,
				Value: val,
			})
		} else if utils.IsInStringArray(k, metrics) {
			metricPairs = append(metricPairs, influxdb.SKeyValue{
				Key: k, Value: val,
			})
		}
	}
	metric.Tags = tagPairs
	metric.Metrics = metricPairs
	return metric, nil
}

func SendMetrics(s *mcclient.ClientSession, metrics []influxdb.SMetricData, debug bool, database string) error {
	urls, err := s.GetServiceURLs("influxdb", o.Options.SessionEndpointType)
	if err != nil {
		return errors.Wrap(err, "GetServiceURLs")
	}
	if len(database) == 0 {
		database = o.Options.InfluxDatabase
	}
	return influxdb.SendMetrics(urls, database, metrics, debug)
}

func ReportCloudMetricOfoperatorType(operatorType string, session *mcclient.ClientSession, args *ReportOptions) error {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), KEY_LIMIT)
	query.Add(jsonutils.NewString("true"), KEY_ADMIN)
	//query.Add(jsonutils.NewString("true"), KEY_USABLE)
	if len(args.Provider) > 0 {
		for _, val := range args.Provider {
			query.Add(jsonutils.NewString(val), "provider")
		}
	}
	cloudProviderList, err := (&modules.Cloudproviders).List(session, query)
	if err != nil {
		return errors.Wrap(err, "cloudProviders get list error")
	}
	providerGroup, _ := errgroup.WithContext(context.Background())
	tmpCount := 0
	if args.Count == 0 {
		args.Count = 1
	}
	for i := 0; i < len(cloudProviderList.Data); i++ {
		provider := cloudProviderList.Data[i]
		status, err := provider.GetString("status")
		if err != nil {
			return errors.Wrap(err, "provider get status error")
		}
		if status == "connected" {
			providerStruct := SProvider{}
			err := provider.Unmarshal(&providerStruct)
			if err != nil {
				return errors.Wrap(err, "provider.Unmarshal")
			}
			err = (&providerStruct).Validate()
			if err != nil {
				return errors.Wrap(err, "provider Invalidate")
			}
			providerStr := providerStruct.Provider
			cloudReportFactory, err := GetCloudReportFactory(providerStr)
			if err != nil {
				log.Errorln(errors.Wrap(err, "GetCloudReportFactory"))
				continue
			}
			tmpCount++
			providerGroup.Go(func() error {
				err = cloudReportFactory.NewCloudReport(&providerStruct, session, args, operatorType).Report()
				if err != nil {
					log.Errorln(errors.Wrap(err, "cloudReport Report method"))
				}
				return nil
			})
			if tmpCount == args.Count {
				err := providerGroup.Wait()
				if err != nil {
					return err
				}
				tmpCount = 0
			}
		}
	}
	return providerGroup.Wait()
}

func newGroupFunc(func()) {

}

func ReportCustomizeCloudMetric(operatorType string, session *mcclient.ClientSession, args *ReportOptions) error {
	cloudReportFactory, err := GetCloudReportFactory(operatorType)
	if err != nil {
		return errors.Wrap(err, "GetCloudReportFactory")
	}
	err = cloudReportFactory.NewCloudReport(nil, session, args, operatorType).Report()
	if err != nil {
		return errors.Wrap(err, "cloudReport Report method")
	}
	return nil
}

func CollectRegionMetricAsync(asynCount int, region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject, report ICloudReport) error {
	metricGroup, _ := errgroup.WithContext(context.Background())
	count := 0
	if asynCount == 0 {
		asynCount = 10
	}
	for i, _ := range servers {
		server := servers[i]
		metricGroup.Go(func() error {
			return report.CollectRegionMetric(region, []jsonutils.JSONObject{server})
		})
		count++
		if count == asynCount {
			err := metricGroup.Wait()
			if err != nil {
				return err
			}
			count = 0
		}
	}
	return metricGroup.Wait()
}
