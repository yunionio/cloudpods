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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	o "yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func jsonToMetric(obj *jsonutils.JSONDict, name string, tags []string, metrics []string) (influxdb.SMetricData, error) {
	metric := influxdb.SMetricData{Name: name}
	objMap, err := obj.GetMap()
	if err != nil {
		return metric, errors.Wrap(err, "obj.GetMap")
	}
	tagPairs := make([]influxdb.SKeyValue, 0)
	metricPairs := make([]influxdb.SKeyValue, 0)
	for k, v := range objMap {
		val, _ := v.GetString()
		if utils.IsInStringArray(k, tags) {
			tagPairs = append(tagPairs, influxdb.SKeyValue{
				Key: k, Value: val,
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

func sendMetrics(s *mcclient.ClientSession, metrics []influxdb.SMetricData, debug bool) error {
	urls, err := s.GetServiceURLs("influxdb", o.Options.SessionEndpointType)
	if err != nil {
		return errors.Wrap(err, "GetServiceURLs")
	}
	return influxdb.SendMetrics(urls, o.Options.InfluxDatabase, metrics, debug)
}

type TListFunc func(*mcclient.ClientSession, jsonutils.JSONObject) (*modulebase.ListResult, error)
type TProcessFunc func(jsonutils.JSONObject) error

func listAll(s *mcclient.ClientSession, listFunc TListFunc, kwargs jsonutils.JSONObject, processFunc TProcessFunc) error {
	type sListParams struct {
		Limit  int
		Offset int
	}
	offset := 0
	total := -1
	for total < 0 || offset < total {
		params := jsonutils.Marshal(sListParams{
			Limit:  100,
			Offset: offset,
		})
		if kwargs != nil {
			params.(*jsonutils.JSONDict).Update(kwargs)
		}
		result, err := listFunc(s, jsonutils.Marshal(params))
		if err != nil {
			return err
		}
		total = result.Total
		for i := range result.Data {
			offset += 1
			err = processFunc(result.Data[i])
			if err != nil {
				log.Errorf("fail to processData %s: %s", result.Data[i], err)
			}
		}
	}
	return nil
}
