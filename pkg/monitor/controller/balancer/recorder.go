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

package balancer

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/wait"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	computemod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type EventAction string

const (
	EventActionFindResultFail = "find_result_fail"
	EventActionMigrating      = "migrating"
	EventActionMigrateSuccess = "migrate_success"
	EventActionMigrateFail    = "migrate_fail"
	EventActionMigrateError   = "migrate_error"
)

type IRecorder interface {
	Record(userCred mcclient.TokenCredential, alert *models.SMigrationAlert, notes interface{}, act EventAction)
	RecordError(userCred mcclient.TokenCredential, alert *models.SMigrationAlert, err error, act EventAction)

	RecordMigrate(ctx context.Context, s *mcclient.ClientSession, alert *models.SMigrationAlert, note *models.MigrateNote) error
	RecordMigrateError(userCred mcclient.TokenCredential, alert *models.SMigrationAlert, note *models.MigrateNote, err error) error

	StartWatchMigratingProcess(ctx context.Context, s *mcclient.ClientSession, alert *models.SMigrationAlert, note *models.MigrateNote)
}

type sRecorder struct{}

func NewRecorder() IRecorder {
	return new(sRecorder)
}

func (r *sRecorder) Record(userCred mcclient.TokenCredential, alert *models.SMigrationAlert, notes interface{}, act EventAction) {
	db.OpsLog.LogEvent(alert, string(act), notes, userCred)
}

func (r *sRecorder) RecordError(userCred mcclient.TokenCredential, alert *models.SMigrationAlert, err error, act EventAction) {
	db.OpsLog.LogEvent(alert, string(act), err, userCred)
}

func NewMigrateNote(pair *resultPair, err error) (*models.MigrateNote, error) {
	gst := new(models.MigrateNoteGuest)
	src := pair.source
	if err := src.GetObject().Unmarshal(gst); err != nil {
		return nil, errors.Wrap(err, "Unmarshal source")
	}
	gst.Host = src.GetHostName()
	gst.Score = pair.source.GetScore()

	target := pair.target
	host := &models.MigrateNoteTarget{
		Id:    target.GetId(),
		Name:  target.GetName(),
		Score: target.GetCurrent(),
	}

	note := &models.MigrateNote{
		Guest:  gst,
		Target: host,
	}
	if err != nil {
		note.Error = err.Error()
	}
	return note, nil
}

func (r *sRecorder) RecordMigrate(ctx context.Context, s *mcclient.ClientSession, alert *models.SMigrationAlert, note *models.MigrateNote) error {
	r.Record(s.GetToken(), alert, note, EventActionMigrating)
	if err := alert.SetMigrateNote(ctx, note, false); err != nil {
		return errors.Wrap(err, "SetMigrateNote")
	}
	// start watcher to trace migrating process
	r.StartWatchMigratingProcess(ctx, s, alert, note)
	return nil
}

func (r *sRecorder) StartWatchMigratingProcess(ctx context.Context, s *mcclient.ClientSession, alert *models.SMigrationAlert, note *models.MigrateNote) {
	go r.startWatchMigratingProcess(ctx, s, alert, note)
}

func (r *sRecorder) startWatchMigratingProcess(ctx context.Context, s *mcclient.ClientSession, alert *models.SMigrationAlert, note *models.MigrateNote) {
	interval := time.Second * 30
	noteStr := jsonutils.Marshal(note).String()
	serverId := note.Guest.Id
	serverName := note.Guest.Name
	sourceHostId := note.Guest.HostId
	targetHostId := note.Target.Id
	if err := wait.PollImmediateInfinite(interval, func() (bool, error) {
		log.Infof("start to watch migrating process %s", noteStr)
		srvObj, err := computemod.Servers.Get(s, serverId, jsonutils.NewDict())
		if err != nil {
			return false, errors.Wrapf(err, "Get server %s(%s) from cloud", serverName, serverId)
		}
		status, err := srvObj.GetString("status")
		if err != nil {
			return false, errors.Wrapf(err, "Get server status %s(%s)", serverName, serverId)
		}
		curHostId, err := srvObj.GetString("host_id")
		if err != nil {
			return false, errors.Wrapf(err, "Get server host_id %s(%s)", serverName, serverId)
		}
		if utils.IsInStringArray(status, []string{compute.VM_RUNNING, compute.VM_READY}) {
			if curHostId != sourceHostId {
				if curHostId == targetHostId {
					return true, nil
				} else {
					return false, errors.Errorf("Server expected in target host %s, current %s", targetHostId, curHostId)
				}
			} else {
				return false, errors.Errorf("Server still in source host %s", sourceHostId)
			}
		} else if strings.HasSuffix(status, "_fail") || strings.HasSuffix(status, "_failed") {
			if status == compute.VM_MIGRATE_FAILED {
				// try sync status to orignal
				if _, err := computemod.Servers.PerformAction(s, serverId, "syncstatus", nil); err != nil {
					log.Errorf("Sync server %s(%s) migrate_failed status error: %v", serverName, serverId, err)
				}
			}
			return false, errors.Errorf("Server fail status %s", status)
		}
		log.Infof("%s: server status %q, continue watching", noteStr, status)
		return false, nil
	}); err != nil {
		note.Error = err.Error()
		r.Record(s.GetToken(), alert, note, EventActionMigrateFail)
		if err := alert.SetMigrateNote(ctx, note, true); err != nil {
			log.Errorf("Delete alert %s(%s) migrate note on failure: %s", alert.GetName(), alert.GetId(), noteStr)
		}
	} else {
		r.Record(s.GetToken(), alert, note, EventActionMigrateSuccess)
		man := models.MonitorResourceManager
		man.SyncManually(ctx)
		if err := alert.SetMigrateNote(ctx, note, true); err != nil {
			log.Errorf("Delete alert %s(%s) migrate note on success: %s", alert.GetName(), alert.GetId(), noteStr)
		}
	}
}

func (r *sRecorder) RecordMigrateError(userCred mcclient.TokenCredential, alert *models.SMigrationAlert, note *models.MigrateNote, err error) error {
	note.Error = err.Error()
	r.Record(userCred, alert, note, EventActionMigrateError)
	return nil
}
