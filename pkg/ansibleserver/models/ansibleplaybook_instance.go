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
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/ansibleserver/options"
	api "yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/workmanager"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/ansiblev2"
)

type SAnsiblePlaybookInstance struct {
	db.SStatusStandaloneResourceBase
	ReferenceId string `width:"36" nullable:"false" get:"user" list:"user"`
	Inventory   string `length:"text" nullable:"false" get:"user" list:"user"`
	Params      jsonutils.JSONObject
	Output      string    `length:"medium" get:"user" list:"user"`
	StartTime   time.Time `list:"user" get:"user"`
	EndTime     time.Time `list:"user" get:"user"`
}

type SAnsiblePlaybookInstanceManager struct {
	db.SStatusStandaloneResourceBaseManager

	sessions    ansible.SessionManager
	sessionsMux *sync.Mutex
}

var AnsiblePlaybookInstanceManager *SAnsiblePlaybookInstanceManager

func init() {
	AnsiblePlaybookInstanceManager = &SAnsiblePlaybookInstanceManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAnsiblePlaybookInstance{},
			"ansibleplaybook_instance_tbl",
			"ansibleplaybookinstance",
			"ansibleplaybookinstances",
		),
		sessions: ansible.SessionManager{},
	}
	AnsiblePlaybookInstanceManager.SetVirtualObject(AnsiblePlaybookInstanceManager)
}

func (aim *SAnsiblePlaybookInstanceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.AnsiblePlaybookInstanceListInput) (*sqlchemy.SQuery, error) {
	if len(input.AnsiblePlayboookReferenceId) > 0 {
		q = q.Equals("reference_id", input.AnsiblePlayboookReferenceId)
	}
	return aim.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
}

func (aim *SAnsiblePlaybookInstanceManager) createInstance(ctx context.Context, referenceId string, host api.AnsibleHost, params jsonutils.JSONObject) (*SAnsiblePlaybookInstance, error) {
	// build inventory
	inv := ansiblev2.NewInventory()
	vars := map[string]interface{}{
		"ansible_user": host.User,
		"ansible_host": host.IP,
		"ansible_port": host.Port,
	}
	h := ansiblev2.NewHost()
	h.Vars = vars
	inv.SetHost(host.Name, h)
	ai := &SAnsiblePlaybookInstance{
		ReferenceId: referenceId,
		Params:      params,
		Inventory:   inv.String(),
	}
	err := aim.TableSpec().Insert(ctx, ai)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create AnsiblePlaybookInstance")
	}
	ai.SetModelManager(aim, ai)
	return ai, nil
}

func (ai *SAnsiblePlaybookInstance) AllowPerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (ai *SAnsiblePlaybookInstance) PerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, ai.runPlaybook(ctx, userCred, nil)
}

func (ai *SAnsiblePlaybookInstance) runPlaybook(ctx context.Context, userCred mcclient.TokenCredential, ar *SAnsiblePlaybookReference) error {
	man := AnsiblePlaybookInstanceManager
	if man.sessions.Has(ai.Id) {
		return errors.Error("playbook is already running")
	}

	if ar == nil {
		obj, err := AnsiblePlaybookReferenceManager.FetchById(ai.ReferenceId)
		if err != nil {
			return errors.Wrapf(err, "unable to fetch ansibleplaybook reference %s", ai.ReferenceId)
		}
		ar = obj.(*SAnsiblePlaybookReference)
	}
	var (
		privateKey string
		err        error
	)
	if privateKey, err = modules.Sshkeypairs.FetchPrivateKey(ctx, userCred); err != nil {
		return err
	}
	_, err = db.Update(ai, func() error {
		ai.StartTime = time.Now()
		ai.EndTime = time.Time{}
		ai.Output = ""
		ai.Status = api.AnsiblePlaybookStatusRunning
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "unable to update ansibleplaybookinstance")
	}

	convertJO := func(o jsonutils.JSONObject) map[string]interface{} {
		if o == nil {
			return map[string]interface{}{}
		}
		ret := make(map[string]interface{})
		o.Unmarshal(&ret)
		return ret
	}
	// merge configs
	params := make(map[string]interface{})
	for k, v := range convertJO(ar.DefaultParams) {
		params[k] = v
	}
	for k, v := range convertJO(ai.Params) {
		params[k] = v
	}
	sess := ansiblev2.NewOfflineSession().
		Inventory(ai.Inventory).
		PrivateKey(privateKey).
		Configs(params).
		PlaybookPath(ar.PlaybookPath).
		OutputWriter(&ansiblePlaybookOutputWriter{ai}).
		KeepTmpdir(options.Options.KeepTmpdir)

	man.sessions.Add(ai.Id, sess)

	// NOTE host state check? run only on online hosts and running guests, skip others
	run := func(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
		defer func() {
			man.sessions.Remove(ai.Id)
		}()
		runErr := man.sessions.Run(ai.Id)
		// TODO: try to close local forwarding?

		_, err := db.Update(ai, func() error {
			err := man.sessions.Err(ai.Id)
			if err != nil {
				ai.Status = api.AnsiblePlaybookStatusCanceled
			} else if runErr != nil {
				log.Warningf("playbook %s(%s) failed: %v", ai.Name, ai.Id, runErr)
				ai.Status = api.AnsiblePlaybookStatusFailed
			} else {
				ai.Status = api.AnsiblePlaybookStatusSucceeded
			}
			ai.EndTime = time.Now()
			return nil
		})
		if err != nil {
			log.Errorf("updating ansible playbook failed: %v", err)
		}
		return nil, runErr
	}
	PlaybookWorker.DelayTask(ctx, run, nil)
	return nil
}

func (ai *SAnsiblePlaybookInstance) stopPlaybook(ctx context.Context, userCred mcclient.TokenCredential) error {
	man := AnsiblePlaybookInstanceManager
	if !man.sessions.Has(ai.Id) {
		return errors.Error("playbook is not running")
	}
	// the playbook will be removed from session map in runPlaybook() on return from run
	man.sessions.Stop(ai.Id)
	return nil
}

func (ai *SAnsiblePlaybookInstance) getMaxOutputLength() int {
	return OutputMaxBytes
}

func (ai *SAnsiblePlaybookInstance) getOutput() string {
	return ai.Output
}

func (ai *SAnsiblePlaybookInstance) setOutput(s string) {
	ai.Output = s
}

var PlaybookWorker *workmanager.SWorkManager

func taskFailed(ctx context.Context, reason string) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		session := auth.GetAdminSessionWithInternal(ctx, "", "")
		modules.TaskFailed(&modules.DevtoolTasks, session, taskId.(string), reason)
	} else {
		log.Warningf("Reqeuest task failed missing task id, with reason: %s", reason)
	}
}

func taskCompleted(ctx context.Context, data jsonutils.JSONObject) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		session := auth.GetAdminSessionWithInternal(ctx, "", "")
		modules.TaskComplete(&modules.DevtoolTasks, session, taskId.(string), data)
	} else {
		log.Warningf("Reqeuest task failed missing task id, with data: %v", data)
	}
}

func InitPlaybookWorker() {
	PlaybookWorker = workmanager.NewWorkManger(taskFailed, taskCompleted, options.Options.PlaybookWorkerCount)
}
