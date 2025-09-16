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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	baseOptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func StatusProbe(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if options.Options.EnableStatusProbeDebug {
		log.Debugf("Start resource status probe")
	}

	if !options.Options.EnableStatusProbe {
		if options.Options.EnableStatusProbeDebug {
			log.Debugf("Resource status probe is disabled")
		}
		return
	}

	sess := auth.GetSession(ctx, userCred, options.Options.Region)
	metrics := make([]influxdb.SMetricData, 0)
	for _, model := range options.Options.StatusProbeModels {
		mts, err := doModelStatusProbe(sess, model)
		if err != nil {
			log.Errorf("doModelStatusProbe failed: %s", err)
		}
		metrics = append(metrics, mts...)
	}

	err := sendMetrics(sess, metrics, "system")
	if err != nil {
		log.Errorf("StatusProbe SendMetrics error: %s", err)
	}
}

func doModelStatusProbe(sess *mcclient.ClientSession, modelName string) ([]influxdb.SMetricData, error) {
	model, err := modulebase.GetModule(sess, modelName)
	if err != nil {
		return nil, errors.Wrap(err, "GetModule")
	}

	listOpts := baseOptions.BaseListOptions{}
	listOpts.Scope = "max"
	listOpts.SummaryStats = true
	limit := 0
	listOpts.Limit = &limit

	results, err := model.List(sess, jsonutils.Marshal(listOpts))
	if err != nil {
		return nil, errors.Wrap(err, "List")
	}

	statusInfoTotal := struct {
		apis.TotalCountBase
		StatusInfo []apis.StatusStatisticStatusInfo
	}{}

	err = results.Totals.Unmarshal(&statusInfoTotal)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal statusInfoTotal")
	}

	log.Infof("statusInfoTotal: %s", jsonutils.Marshal(statusInfoTotal))

	metrics := make([]influxdb.SMetricData, 0)

	totalCount := int64(0)
	pendingDeletedCount := int64(0)
	for _, statusInfo := range statusInfoTotal.StatusInfo {
		metrics = append(metrics, genStatusMetricData(model, statusInfo.Status, statusInfo.TotalCount, statusInfo.PendingDeletedCount))
		totalCount += statusInfo.TotalCount
		pendingDeletedCount += statusInfo.PendingDeletedCount
	}
	metrics = append(metrics, genStatusMetricData(model, "total", totalCount, pendingDeletedCount))

	if options.Options.EnableStatusProbeDebug {
		log.Debugf("StatusProbe for model %s metrics: %s", modelName, jsonutils.Marshal(metrics))
	}

	return metrics, nil
}

func genStatusMetricData(model modulebase.Manager, status string, count int64, pendingDeletedCount int64) influxdb.SMetricData {
	return influxdb.SMetricData{
		Name: "status_probe",
		Tags: influxdb.TKeyValuePairs{
			influxdb.SKeyValue{
				Key:   "service",
				Value: model.ServiceType(),
			},
			influxdb.SKeyValue{
				Key:   "model",
				Value: model.GetKeyword(),
			},
			influxdb.SKeyValue{
				Key:   "status",
				Value: status,
			},
		},
		Metrics: influxdb.TKeyValuePairs{
			influxdb.SKeyValue{
				Key:   "count",
				Value: fmt.Sprintf("%d", count),
			},
			influxdb.SKeyValue{
				Key:   "pending_deleted",
				Value: fmt.Sprintf("%d", pendingDeletedCount),
			},
		},
	}
}
