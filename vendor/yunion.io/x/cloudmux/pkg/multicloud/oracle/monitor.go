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

package oracle

import (
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SOracleClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SOracleClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	query := url.Values{}
	query.Set("compartmentId", self.compartment)

	results := []cloudprovider.MetricValues{}
	for metricType, queryStr := range map[cloudprovider.TMetricType]string{
		cloudprovider.VM_METRIC_TYPE_CPU_USAGE:          fmt.Sprintf("CPUUtilization[1m]{resourceId = \"%s\"}.mean()", opts.ResourceId),
		cloudprovider.VM_METRIC_TYPE_MEM_USAGE:          fmt.Sprintf("MemoryUtilization[1m]{resourceId = \"%s\"}.mean()", opts.ResourceId),
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:  fmt.Sprintf("DiskIopsRead[1m]{resourceId = \"%s\"}.rate()", opts.ResourceId),
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS: fmt.Sprintf("DiskIopsWritten[1m]{resourceId = \"%s\"}.rate()", opts.ResourceId),
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:   fmt.Sprintf("DiskBytesRead[1m]{resourceId = \"%s\"}.rate()", opts.ResourceId),
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:  fmt.Sprintf("DiskBytesWritten[1m]{resourceId = \"%s\"}.rate()", opts.ResourceId),
		cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:         fmt.Sprintf("NetworksBytesIn[1m]{resourceId = \"%s\"}.rate()", opts.ResourceId),
		cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:         fmt.Sprintf("NetworksBytesOut[1m]{resourceId = \"%s\"}.rate()", opts.ResourceId),
	} {
		params := map[string]interface{}{
			"namespace": "oci_computeagent",
			"startTime": opts.StartTime.UTC().Format(time.RFC3339),
			"endTime":   opts.EndTime.UTC().Format(time.RFC3339),
			"query":     queryStr,
		}
		resp, err := self.post(SERVICE_TELEMETRY, opts.RegionExtId, "metrics/actions/summarizeMetricsData", query, params)
		if err != nil {
			return nil, err
		}
		ret := []struct {
			AggregatedDatapoints []struct {
				Timestamp time.Time
				Value     float64
			}
		}{}
		err = resp.Unmarshal(&ret)
		if err != nil {
			return nil, err
		}
		result := cloudprovider.MetricValues{
			Id:         opts.ResourceId,
			MetricType: metricType,
			Values:     []cloudprovider.MetricValue{},
		}
		for _, values := range ret {
			for _, v := range values.AggregatedDatapoints {
				result.Values = append(result.Values, cloudprovider.MetricValue{
					Timestamp: v.Timestamp,
					Value:     v.Value,
				})
			}
		}
		results = append(results, result)

	}
	return results, nil
}
