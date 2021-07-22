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

package collectors

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

var config map[string]string = map[string]string{
	"hypervisors":           "host-type",
	"compute_engine_brands": "provider",
}
var measureMent string = "usage"

func init() {
	shellutils.R(&common.ReportOptions{}, "report-usage", "Report Usage", reportUsage)
}

func reportUsage(session *mcclient.ClientSession, args *common.ReportOptions) error {
	dataList := make([]influxdb.SMetricData, 0)
	nowTime := time.Now()
	//镜像使用量信息
	imageUsageFields, err := getImageUsageFields(session)
	if err != nil {
		return err
	}
	//查询到的Usage信息统一放置在metric中
	imageUsageFieldsDict := imageUsageFields.(*jsonutils.JSONDict)
	capabilitesQuery := jsonutils.NewDict()
	capabilitesQuery.Add(jsonutils.NewString("system"), "scope")
	capabilites, err := modules.Capabilities.List(session, capabilitesQuery)
	if err != nil {
		return err
	}
	//通过capabilities中的信息遍历hypevisors和brands
	for i := 0; i < len(capabilites.Data); i++ {
		capabilitesObj := capabilites.Data[i]
		capDict, ok := capabilitesObj.(*jsonutils.JSONDict)
		if !ok {
			return errors.ErrClient
		}
		for _, capKey := range capDict.SortedKeys() {
			if _, ok := config[capKey]; ok {
				hypeOrBrandObj, _ := capDict.Get(capKey)
				if hypeOrBrandObj != nil {
					hypeOrBrandArr, _ := hypeOrBrandObj.(*jsonutils.JSONArray)
					for i := 0; i < len(hypeOrBrandArr.Value()); i++ {
						hypeOrBrand := hypeOrBrandArr.Value()[i].(*jsonutils.JSONString)
						dataList, err = packMetricList(session, dataList, imageUsageFieldsDict, config[capKey],
							hypeOrBrand.String(), nowTime)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	//查询host-type==""的情况，对应onecloud-控制面板-全部 要展示的内容
	dataList, err = packMetricList(session, dataList, imageUsageFieldsDict, "host-type", "", nowTime)
	//写入influDb
	return sendMetrics(session, dataList, args.Debug)
}

//根据capabilities中的hypevisors和brands中的对应属性，组装Metric
func packMetricList(session *mcclient.ClientSession, dataList []influxdb.SMetricData,
	imageUsageFieldsDict *jsonutils.JSONDict, paramKey string,
	paramValue string, nowTime time.Time) (rtnList []influxdb.SMetricData, err error) {
	//query，sql信息，查询主机compute的使用信息
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("system"), "scope")
	if paramValue != "" {
		if paramKey == "host-type" {
			if paramValue == "kvm" {
				query.Add(jsonutils.NewString("hypervisor"), paramKey)
			}
			query.Add(jsonutils.NewString(paramValue), paramKey)
		}
		query.Add(jsonutils.NewString(paramValue), paramKey)
	}
	metric := &influxdb.SMetricData{Name: measureMent, Timestamp: nowTime}
	//查询到的镜像的使用信息放到SMetricData中的metric
	metric, _ = jsonTometricData(imageUsageFieldsDict, metric, "metric")
	//compute主机使用量
	respObj, err := modules.Usages.GetGeneralUsage(session, query)
	if err != nil {
		return nil, err
	}
	respObjDict := respObj.(*jsonutils.JSONDict)
	//查询到的主机的使用信息放到SMetricData中的metric
	metric, _ = jsonTometricData(respObjDict, metric, "metric")
	if paramValue != "" {
		metric.Tags = append(metric.Tags, influxdb.SKeyValue{
			Key: paramKey, Value: paramValue,
		})
	} else {
		metric.Tags = append(metric.Tags, influxdb.SKeyValue{
			Key: paramKey, Value: "all",
		})
	}
	dataList = append(dataList, *metric)
	return dataList, nil
}

//获得镜像使用量
func getImageUsageFields(session *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	respObj, e := (&modules.ImageUsages).GetUsage(session, nil)
	if e != nil {
		return nil, e
	}
	respDict, ok := respObj.(*jsonutils.JSONDict)
	if !ok {
		return nil, jsonutils.ErrInvalidJsonDict
	}
	return respDict, nil
}

//将JSONDict的信息放置到SMetricData中
func jsonTometricData(obj *jsonutils.JSONDict, metric *influxdb.SMetricData,
	metricDataType string) (*influxdb.SMetricData, error) {

	objMap, err := obj.GetMap()
	if err != nil {
		return nil, errors.Wrap(err, "obj.GetMap")
	}
	tagPairs := make([]influxdb.SKeyValue, 0)
	metricPairs := make([]influxdb.SKeyValue, 0)
	for k, v := range objMap {
		val, _ := v.GetString()
		if metricDataType == "tag" {
			tagPairs = append(tagPairs, influxdb.SKeyValue{
				Key: k, Value: val,
			})
		} else if metricDataType == "metric" {
			metricPairs = append(metricPairs, influxdb.SKeyValue{
				Key: k, Value: val,
			})
		}
	}
	metric.Tags = append(metric.Tags, tagPairs...)
	metric.Metrics = append(metric.Metrics, metricPairs...)
	return metric, nil
}
