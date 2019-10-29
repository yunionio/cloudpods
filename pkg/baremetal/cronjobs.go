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

package baremetal

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/redfish"
)

type IBaremetalCronJob interface {
	Name() string
	NeedsToRun(now time.Time) bool
	StartRun()
	Do(ctx context.Context, now time.Time) error
	StopRun()
}

func DoCronJobs(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		return
	}
	for _, bm := range GetBaremetalManager().GetBaremetals() {
		if bm.GetTask() == nil {
			bm.doCronJobs(ctx)
		}
	}
}

type SBaseBaremetalCronJob struct {
	lastTime  time.Time
	running   bool
	interval  time.Duration
	baremetal *SBaremetalInstance
}

func (job *SBaseBaremetalCronJob) NeedsToRun(now time.Time) bool {
	if (job.lastTime.IsZero() || now.Sub(job.lastTime) > job.interval) && !job.running {
		return true
	}
	return false
}

func (job *SBaseBaremetalCronJob) StartRun() {
	job.running = true
}

func (job *SBaseBaremetalCronJob) StopRun() {
	job.running = false
}

type SStatusProbeJob struct {
	SBaseBaremetalCronJob
}

func NewStatusProbeJob(baremetal *SBaremetalInstance, interval time.Duration) IBaremetalCronJob {
	return &SStatusProbeJob{
		SBaseBaremetalCronJob: SBaseBaremetalCronJob{
			baremetal: baremetal,
			interval:  interval,
		},
	}
}

func (job *SStatusProbeJob) Name() string {
	return "StatusProbeJob"
}

func (job *SStatusProbeJob) Do(ctx context.Context, now time.Time) error {
	bStatus := job.baremetal.GetStatus()
	if bStatus == api.BAREMETAL_READY || bStatus == api.BAREMETAL_RUNNING {
		ps, err := job.baremetal.GetPowerStatus()
		if err != nil {
			return errors.Wrap(err, "GetPowerStatus")
		}
		job.lastTime = now
		pps := PowerStatusToBaremetalStatus(ps)
		if pps != bStatus {
			log.Debugf("Detected baremetal status change!")
			job.baremetal.SyncAllStatus(ps)
			return nil
		}
	}
	return nil
}

type SLogFetchJob struct {
	SBaseBaremetalCronJob
}

func NewLogFetchJob(baremetal *SBaremetalInstance, interval time.Duration) IBaremetalCronJob {
	return &SLogFetchJob{
		SBaseBaremetalCronJob: SBaseBaremetalCronJob{
			baremetal: baremetal,
			interval:  interval,
		},
	}
}

func (job *SLogFetchJob) Name() string {
	return "LogFetchJob"
}

func eventToJson(e redfish.SEvent) *jsonutils.JSONDict {
	json := jsonutils.NewDict()
	json.Add(jsonutils.NewString(timeutils.IsoTime(e.Created)), "created")
	json.Add(jsonutils.NewString(e.EventId), "event_id")
	json.Add(jsonutils.NewString(e.Message), "message")
	json.Add(jsonutils.NewString(e.Severity), "severity")
	json.Add(jsonutils.NewString(e.Type), "type")
	return json
}

func fetchLogs(baremetal *SBaremetalInstance, ctx context.Context, logType string) error {
	params := jsonutils.NewDict()
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "")
	params.Add(jsonutils.NewInt(1), "limit")
	params.Add(jsonutils.NewString(baremetal.GetId()), "host_id")
	params.Add(jsonutils.NewString(logType), "type")
	result, err := modules.BaremetalEvents.List(s, params)
	if err != nil {
		return errors.Wrap(err, "modules.BaremetalEvents.List")
	}
	log.Debugf("%s", result.Data)
	var eventId string
	var since time.Time
	if len(result.Data) > 0 {
		since, _ = result.Data[0].GetTime("created")
		eventId, _ = result.Data[0].GetString("event_id")
	}
	offset := 0
	logs, err := baremetal.fetchLogs(ctx, logType, since)
	if err != nil {
		return errors.Wrap(err, "job.baremetal.FetchLogs")
	}
	if len(eventId) > 0 {
		// latset eventId
		for i := range logs {
			if logs[i].EventId == eventId {
				offset = i + 1
				break
			}
		}
	}
	for i := offset; i < len(logs); i += 1 {
		eventData := eventToJson(logs[i])
		eventData.Add(jsonutils.NewString(baremetal.GetId()), "host_id")
		eventData.Add(jsonutils.NewString(baremetal.GetName()), "host_name")
		eventData.Add(jsonutils.NewString(baremetal.GetIPMINicIPAddr()), "ipmi_ip")
		log.Debugf("%s", eventData)
		_, err := modules.BaremetalEvents.Create(s, eventData)
		if err != nil {
			return errors.Wrap(err, "modules.BaremetalEvents.Create")
		}
	}
	return nil
}

func (job *SLogFetchJob) Do(ctx context.Context, now time.Time) error {
	if !job.baremetal.isRedfishCapable() {
		return nil
	}
	err := fetchLogs(job.baremetal, ctx, redfish.EVENT_TYPE_SYSTEM)
	if err != nil {
		return errors.Wrap(err, "fetchLogs api.EVENT_TYPE_SYSTEM")
	}
	err = fetchLogs(job.baremetal, ctx, redfish.EVENT_TYPE_MANAGER)
	if err != nil {
		return errors.Wrap(err, "fetchLogs api.EVENT_TYPE_MANAGER")
	}
	job.lastTime = now
	return nil
}

type SSendMetricsJob struct {
	SBaseBaremetalCronJob
}

func NewSendMetricsJob(baremetal *SBaremetalInstance, interval time.Duration) IBaremetalCronJob {
	return &SSendMetricsJob{
		SBaseBaremetalCronJob: SBaseBaremetalCronJob{
			baremetal: baremetal,
			interval:  interval,
		},
	}
}

func (job *SSendMetricsJob) Name() string {
	return "SendMetricsJob"
}

func (job *SSendMetricsJob) Do(ctx context.Context, now time.Time) error {
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "")
	urls, err := s.GetServiceURLs("influxdb", o.Options.SessionEndpointType)
	if err != nil {
		return errors.Wrap(err, "s.GetServiceURLs")
	}
	if len(urls) == 0 {
		return nil
	}
	if !job.baremetal.isRedfishCapable() {
		return nil
	}
	powerMetrics, thermalMetrics, err := job.baremetal.fetchPowerThermalMetrics(ctx)
	tags := job.baremetal.getTags()
	metrics := []influxdb.SMetricData{
		{
			Name:      "power",
			Tags:      tags,
			Metrics:   powerMetrics,
			Timestamp: now,
		},
		{
			Name:      "thermal",
			Tags:      tags,
			Metrics:   thermalMetrics,
			Timestamp: now,
		},
	}
	err = influxdb.SendMetrics(urls, "telegraf", metrics, false)
	if err != nil {
		return errors.Wrap(err, "influxdb.SendMetrics")
	}
	job.lastTime = now
	return nil
}
