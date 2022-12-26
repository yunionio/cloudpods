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

package hcs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SCloudVM struct {
	NativeId string
	ResId    string
}

var vmMetric = map[string]cloudprovider.TMetricType{
	"562958543421441": cloudprovider.VM_METRIC_TYPE_CPU_USAGE,
	"562958543486979": cloudprovider.VM_METRIC_TYPE_MEM_USAGE,
	"562958543552537": cloudprovider.VM_METRIC_TYPE_NET_BPS_RX, //kb
	"562958543552538": cloudprovider.VM_METRIC_TYPE_NET_BPS_TX, //kb
	"562958543618052": cloudprovider.VM_METRIC_TYPE_DISK_USAGE,
	"562958543618072": cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS,
	"562958543618073": cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS,
	"562958543618061": cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS,
	"562958543618062": cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS,
}

func (self *SHcsClient) getServerMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	params := url.Values{}
	params.Set("pageNo", "1")
	params.Set("pageSize", "1")
	params.Set("condition", jsonutils.Marshal(map[string]interface{}{
		"constraint": []map[string]interface{}{
			{
				"simple": map[string]interface{}{
					"name":     "nativeId",
					"operator": "equal",
					"value":    opts.ResourceId,
				},
			},
		},
	}).String())
	resp, err := self.ocRequest(httputils.GET, "rest/tenant-resource/v1/instances/CLOUD_VM", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "ocRequest")
	}
	vms := []SCloudVM{}
	resp.Unmarshal(&vms, "objList")
	result := []cloudprovider.MetricValues{}
	for i := range vms {
		if vms[i].NativeId != opts.ResourceId {
			continue
		}
		metricIds := []string{}
		for m := range vmMetric {
			metricIds = append(metricIds, m)
		}
		body := map[string]interface{}{
			"obj_type_id":   "562958543355904",
			"indicator_ids": metricIds,
			"obj_ids":       []string{vms[i].ResId},
			"interval":      "MINUTE",
			"range":         "BEGIN_END_TIME",
			"begin_time":    opts.StartTime.Unix() * 1000,
			"end_time":      opts.EndTime.Unix() * 1000,
		}
		resp, err := self.ocRequest(httputils.POST, "rest/performance/v1/data-svc/history-data/action/query", nil, body)
		if err != nil {
			return nil, errors.Wrapf(err, "ocRequest")
		}
		if gotypes.IsNil(resp) || !resp.Contains("data") {
			break
		}
		for m, metricType := range vmMetric {
			series := []map[string]string{}
			resp.Unmarshal(&series, "data", vms[i].ResId, m, "series")
			if len(series) == 0 {
				continue
			}
			ret := cloudprovider.MetricValues{
				Id:         opts.ResourceId,
				Values:     []cloudprovider.MetricValue{},
				MetricType: metricType,
			}
			for i := range series {
				for _date, _value := range series[i] {
					date, _ := strconv.Atoi(_date)
					value, _ := strconv.ParseFloat(_value, 32)
					if metricType == cloudprovider.VM_METRIC_TYPE_NET_BPS_RX || metricType == cloudprovider.VM_METRIC_TYPE_NET_BPS_TX {
						value *= 1024
					}
					ret.Values = append(ret.Values, cloudprovider.MetricValue{
						Value:     value,
						Timestamp: time.UnixMilli(int64(date)),
					})
				}
			}
			result = append(result, ret)
		}
	}
	return result, nil
}

func (self *SHcsClient) ocAuth() error {
	url := fmt.Sprintf("https://oc.%s.%s/rest/plat/smapp/v1/oauth/token", self.defaultRegion, self.authUrl)
	client := self.getDefaultClient()
	cli := httputils.NewJsonClient(client)
	params := map[string]interface{}{
		"grantType": "password",
		"userName":  self.account,
		"value":     self.password,
	}
	req := httputils.NewJsonRequest(httputils.PUT, url, jsonutils.Marshal(params))
	_, resp, err := cli.Send(context.Background(), req, &hcsError{Url: url}, self.debug)
	if err != nil {
		return errors.Wrapf(err, "Send")
	}
	self.token, err = resp.GetString("accessSession")
	return errors.Wrapf(err, "resp.accessSession")
}

func (self *SHcsClient) ocRequest(method httputils.THttpMethod, resource string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	self.lock.CheckingLock()
	if len(self.token) == 0 {
		err := self.ocAuth()
		if err != nil {
			return nil, errors.Wrapf(err, "ocAuth")
		}
	}
	client := self.getDefaultClient()
	url := fmt.Sprintf("oc.%s.%s/%s", self.defaultRegion, self.authUrl, resource)
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", url, query.Encode())
	}
	url = strings.TrimPrefix(url, "http://")
	if !strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("https://%s", url)
	}
	var body jsonutils.JSONObject = nil
	if len(params) > 0 {
		body = jsonutils.Marshal(params)
	}
	header := http.Header{}
	header.Set("X-Auth-Token", self.token)
	cli := httputils.NewJsonClient(client)
	req := httputils.NewJsonRequest(method, url, body)
	req.SetHeader(header)
	var resp jsonutils.JSONObject
	var err error
	for i := 0; i < 4; i++ {
		_, resp, err = cli.Send(context.Background(), req, &hcsError{Url: url, Params: params}, self.debug)
		if err == nil {
			break
		}
		if err != nil {
			e, ok := err.(*hcsError)
			if ok {
				if e.Code == 404 {
					return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
				}
				if e.Code == 429 {
					log.Errorf("request %s %v try later", url, err)
					self.lock.Lock()
					time.Sleep(time.Second * 15)
					continue
				}
			}
			return nil, err
		}
	}
	return resp, err
}

func (self *SHcsClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	if !self.isAccountValid() {
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "missing account info")
	}
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.getServerMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.ResourceType)
	}
}
