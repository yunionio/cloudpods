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

package models

import (
	"context"
	"fmt"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SScriptApply struct {
	db.SStatusStandaloneResourceBase
	ScriptId string `width:"36" nullable:"false" index:"true"`
	GuestId  string `width:"36" nullable:"false" index:"true"`
	//
	Args          jsonutils.JSONObject
	TryTimes      int
	ArgsGenerator string `width:"36" nullable:"false"`
}

type SScriptApplyManager struct {
	db.SStatusStandaloneResourceBaseManager
	Session *sScriptApplySession
}

var ScriptApplyManager *SScriptApplyManager

func init() {
	ScriptApplyManager = &SScriptApplyManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SScriptApply{},
			"scriptapply_tbl",
			"scriptapply",
			"scirptapplys",
		),
		Session: newScriptApplySession(),
	}
	ScriptApplyManager.SetVirtualObject(ScriptApplyManager)
}

func (sam *SScriptApplyManager) createScriptApply(ctx context.Context, scriptId, guestId string, args map[string]interface{}, argsGenerator string) (*SScriptApply, error) {
	sa := &SScriptApply{
		ScriptId:      scriptId,
		GuestId:       guestId,
		Args:          jsonutils.Marshal(args),
		ArgsGenerator: argsGenerator,
	}
	err := ScriptApplyManager.TableSpec().Insert(ctx, sa)
	sa.SetModelManager(ScriptApplyManager, sa)
	return sa, err
}

func (sa *SScriptApply) StartApply(ctx context.Context, userCred mcclient.TokenCredential) (err error) {
	if ok := ScriptApplyManager.Session.CheckAndSet(sa.Id); !ok {
		return fmt.Errorf("script %s is applying to server %s", sa.ScriptId, sa.GuestId)
	}
	defer func() {
		if err != nil {
			ScriptApplyManager.Session.Remove(sa.Id)
		}
	}()
	// check try times
	script, err := sa.Script()
	if err != nil {
		return err
	}
	if sa.TryTimes >= script.MaxTryTimes {
		return fmt.Errorf("The times to try has exceeded the maximum times %d setted by the script", script.MaxTryTimes)
	}
	_, err = db.Update(sa, func() error {
		sa.TryTimes += 1
		sa.Status = api.SCRIPT_APPLY_STATUS_APPLYING
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "unable to update scriptapply")
	}

	// The record has to exist before the task is queued. The frontend lists it and
	// opens ansible-log as soon as apply returns, while the task is still waiting
	// to run or still checking SSH.
	sar, err := ScriptApplyRecordManager.CreateRecord(ctx, sa.GetId())
	if err != nil {
		return errors.Wrap(err, "unable to create script apply record")
	}
	err = sa.startApplyScriptTask(ctx, userCred, "", sar.GetId())
	if err != nil {
		failErr := sar.Fail(api.SCRIPT_APPLY_RECORD_FAILCODE_OTHERS, fmt.Sprintf("unable to start ApplyScriptTask: %v", err))
		if failErr != nil {
			return errors.Wrap(failErr, "unable to record")
		}
		ScriptApplyManager.Session.Remove(sa.Id)
		err = nil
	}
	return nil
}

func (sa *SScriptApply) startApplyScriptTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, recordId string) error {
	params := jsonutils.NewDict()
	params.Set("script_apply_record_id", jsonutils.NewString(recordId))
	task, err := taskman.TaskManager.NewTask(ctx, "ApplyScriptTask", sa, userCred, params, "", parentTaskId)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (sa *SScriptApply) StopApply(userCred mcclient.TokenCredential, record *SScriptApplyRecord, success bool, failCode string, reason string) error {
	var status string
	if success {
		status = api.SCRIPT_APPLY_STATUS_READY
		if record != nil {
			record.Succeed(reason)
		}
	} else {
		status = api.SCRIPT_APPLY_RECORD_FAILED
		if record != nil {
			record.Fail(failCode, reason)
		}
	}
	sa.SetStatus(context.Background(), userCred, status, "")
	ScriptApplyManager.Session.Remove(sa.Id)
	return nil
}

func (sa *SScriptApply) Script() (*SScript, error) {
	obj, err := ScriptManager.FetchById(sa.ScriptId)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch Script %s", sa.Id)
	}
	s := obj.(*SScript)
	s.SetModelManager(ScriptManager, s)
	return s, nil
}

type sScriptApplySession struct {
	mux          *sync.Mutex
	applyingOnes sets.String
}

func newScriptApplySession() *sScriptApplySession {
	return &sScriptApplySession{
		mux:          &sync.Mutex{},
		applyingOnes: sets.NewString(),
	}
}

func (sas *sScriptApplySession) CheckAndSet(id string) bool {
	sas.mux.Lock()
	defer sas.mux.Unlock()
	if sas.applyingOnes.Has(id) {
		return false
	}
	sas.applyingOnes.Insert(id)
	return true
}

func (sas *sScriptApplySession) Remove(id string) {
	sas.mux.Lock()
	defer sas.mux.Unlock()
	sas.applyingOnes.Delete(id)
}
