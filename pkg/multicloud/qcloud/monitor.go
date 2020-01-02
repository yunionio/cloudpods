package qcloud

import (
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
)

const (
	QCLOUD_API_VERSION_METRICS = "2018-07-24"
)

type SQcMetricDimension struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type SDataPoint struct {
	Dimensions []SQcMetricDimension `json:"Dimensions"`
	Timestamps []float64            `json:"Timestamps"`
	Values     []float64            `json:"Values"`
}

type SBatchQueryMetricDataInput struct {
	MetricName string               `json:"MetricName"`
	Namespace  string               `json:"Namespace"`
	Metrics    []SQcMetricDimension `json:"Metrics"`
	StartTime  int64                `json:"StartTime"`
	EndTime    int64                `json:"EndTime"`
	Period     string               `json:"Period"`
}

func (r *SRegion) metricsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client := r.GetClient()
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return monitorRequest(cli, action, params, client.Debug)
}

func (r *SRegion) GetMonitorData(name string, ns string, since time.Time, until time.Time,
	demensions []SQcMetricDimension) ([]SDataPoint, error) {
	params := make(map[string]string)
	params["MetricName"] = name
	params["Namespace"] = ns
	if !since.IsZero() {
		params["StartTime"] = since.Format(timeutils.IsoTimeFormat)

	}
	if !until.IsZero() {
		params["EndTime"] = until.Format(timeutils.IsoTimeFormat)
	}
	for index, metricDimension := range demensions {
		i := strconv.FormatInt(int64(index), 10)
		params["Instances."+i+".Dimensions.0.Name"] = metricDimension.Name
		params["Instances."+i+".Dimensions.0.Value"] = metricDimension.Value
	}
	body, err := r.metricsRequest("GetMonitorData", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.MetricRequest")
	}
	dataArray := make([]SDataPoint, 0)
	err = body.Unmarshal(&dataArray, "DataPoints")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return dataArray, nil
}
