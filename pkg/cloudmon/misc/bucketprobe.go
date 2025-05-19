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

package misc

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func BucketProbe(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if options.Options.EnableBucketProbeDebug {
		log.Debugf("BucketProbe start")
	}
	if !options.Options.EnableBucketProbe {
		if options.Options.EnableBucketProbeDebug {
			log.Debugf("BucketProbe is disabled")
		}
		return
	}

	sess := auth.GetSession(ctx, userCred, options.Options.Region)

	metrics, err := gatherBucketMetrics(ctx, sess)
	if err != nil {
		log.Errorf("BucketProbe gatherBucketMetrics failed: %s", err)
		return
	}

	err = sendMetrics(sess, metrics, "telegraf")
	if err != nil {
		log.Errorf("StatusProbe SendMetrics error: %s", err)
	}
}

func gatherBucketMetrics(ctx context.Context, sess *mcclient.ClientSession) ([]influxdb.SMetricData, error) {
	allMetrics := []influxdb.SMetricData{}

	params := baseoptions.BaseListOptions{}
	params.Scope = "max"
	limit := 1000
	params.Limit = &limit
	params.Filter = []string{
		"enable_perf_mon.equals(1)",
	}
	boolTrue := true
	params.Details = &boolTrue

	total := -1
	offset := 0
	for total < 0 || offset < total {
		params.Offset = &offset
		results, err := computemodules.Buckets.List(sess, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrap(err, "computemodules.Buckets.List")
		}
		total = results.Total
		offset = results.Offset + len(results.Data)

		for _, bucket := range results.Data {
			bucketDetails := computeapi.BucketDetails{}
			err = bucket.Unmarshal(&bucketDetails)
			if err != nil {
				log.Errorf("BucketProbe failed: %s", err)
				continue
			}

			metrics, err := probeBucketStats(ctx, sess, &bucketDetails)
			if err != nil {
				log.Errorf("BucketProbe failed: %s", err)
				continue
			}
			allMetrics = append(allMetrics, metrics...)
		}
	}

	return allMetrics, nil
}

func probeBucketStats(ctx context.Context, sess *mcclient.ClientSession, bucketDetails *computeapi.BucketDetails) ([]influxdb.SMetricData, error) {
	bucket, err := computemodules.GetIBucket(ctx, sess, bucketDetails)
	if err != nil {
		return nil, errors.Wrap(err, "getIBucket")
	}

	resultDelay, err := computemodules.ProbeBucketStats(ctx, bucket, options.Options.BucketProbeTestKey, 0)
	if err != nil {
		return nil, errors.Wrap(err, "doProbeBucketStats zero")
	}

	resultRate, err := computemodules.ProbeBucketStats(ctx, bucket, options.Options.BucketProbeTestKey, int64(options.Options.BucketProbeTestSizeMb)*1024*1024)
	if err != nil {
		return nil, errors.Wrap(err, "doProbeBucketStats with payload")
	}

	metricTags := []influxdb.SKeyValue{}
	for k, v := range bucketDetails.GetMetricTags() {
		if len(v) == 0 {
			continue
		}
		metricTags = append(metricTags, influxdb.SKeyValue{
			Key:   k,
			Value: v,
		})
	}

	metrics := []influxdb.SKeyValue{}
	for k, v := range bucketDetails.GetMetricTags() {
		if len(v) == 0 {
			continue
		}
		metrics = append(metrics, influxdb.SKeyValue{
			Key:   k,
			Value: v,
		})
	}

	metrics = append(metrics,
		influxdb.SKeyValue{
			Key:   "upload_delay_ms",
			Value: fmt.Sprintf("%f", resultDelay.UploadDelayMs()),
		},
		influxdb.SKeyValue{
			Key:   "download_delay_ms",
			Value: fmt.Sprintf("%f", resultDelay.DownloadDelayMs()),
		},
		influxdb.SKeyValue{
			Key:   "delete_delay_ms",
			Value: fmt.Sprintf("%f", resultDelay.DeleteDelayMs()),
		},
		influxdb.SKeyValue{
			Key:   "upload_rate_mbps",
			Value: fmt.Sprintf("%f", resultRate.UploadThroughputMbps(options.Options.BucketProbeTestSizeMb)),
		},
		influxdb.SKeyValue{
			Key:   "download_rate_mbps",
			Value: fmt.Sprintf("%f", resultRate.DownloadThroughputMbps(options.Options.BucketProbeTestSizeMb)),
		},
	)

	if options.Options.EnableBucketProbeDebug {
		log.Debugf("BucketProbe for bucket %s metrics: %s", bucketDetails.Name, jsonutils.Marshal(metrics))
	}

	return []influxdb.SMetricData{
		{
			Name:      "bucket_perf",
			Tags:      metricTags,
			Metrics:   metrics,
			Timestamp: time.Now(),
		},
	}, nil
}
