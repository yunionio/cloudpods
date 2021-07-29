package collectors

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BucketStatsOptions struct {
	Debug bool `help:"debug"`
}

func bucketStatsCollect(s *mcclient.ClientSession, args *BucketStatsOptions) error {
	metrics := make([]influxdb.SMetricData, 0)
	listAll(s, modules.Buckets.List, nil,
		func(data jsonutils.JSONObject) error {
			m, err := collectBucket(s, data)
			if err != nil {
				return err
			}
			metrics = append(metrics, m)
			return nil
		},
	)
	return sendMetrics(s, metrics, args.Debug)
}

func collectBucket(s *mcclient.ClientSession, bucket jsonutils.JSONObject) (influxdb.SMetricData, error) {
	metric := influxdb.SMetricData{}
	bucketId, _ := bucket.GetString("id")
	if len(bucketId) == 0 {
		return metric, errors.Error("empty bucket id")
	}
	params := jsonutils.NewDict()
	params.Set("stats_only", jsonutils.JSONTrue)
	result, err := modules.Buckets.PerformAction(s, bucketId, "sync", params)
	if err != nil {
		return metric, errors.Wrap(err, "PerformAction")
	}
	return jsonToMetric(result.(*jsonutils.JSONDict), "bucket",
		[]string{
			"name",
			"id",
			"account",
			"account_id",
			"manager",
			"manager_id",
			"manager_domain",
			"manager_domain_id",
			"manager_project",
			"manager_project_id",
			"brand",
			"provider",
			"region_id",
			"region_ext_id",
			"tenant",
			"tenant_id",
			"domain_id",
			"project_domain",
		},
		[]string{
			"object_cnt",
			"size_bytes",
		},
	)
}

func init() {
	shellutils.R(&BucketStatsOptions{}, "buckets", "Bucket stats", bucketStatsCollect)
}
