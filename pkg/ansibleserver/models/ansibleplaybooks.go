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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/ansibleserver/options"
	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/ansibleserver"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

// low priority
//
// - retry times and interval
// - timeout,
// - copied playbook has no added value

type SAnsiblePlaybook struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase `enabled->default:"" enabled->nullable:"true"`

	Playbook  *ansible.Playbook `length:"text" nullable:"false" create:"required" get:"user" update:"user"`
	Output    string            `length:"medium" get:"user"`
	StartTime time.Time         `list:"user"`
	EndTime   time.Time         `list:"user"`
}

type SAnsiblePlaybookManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager

	sessions    ansible.SessionManager
	sessionsMux *sync.Mutex
}

var AnsiblePlaybookManager *SAnsiblePlaybookManager

func init() {
	AnsiblePlaybookManager = &SAnsiblePlaybookManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAnsiblePlaybook{},
			"ansibleplaybooks_tbl",
			"ansibleplaybook",
			"ansibleplaybooks",
		),
		sessions:    ansible.SessionManager{},
		sessionsMux: &sync.Mutex{},
	}
	AnsiblePlaybookManager.SetVirtualObject(AnsiblePlaybookManager)
}

// requireSystemAdmin is required to enable a playbook.
func requireSystemAdmin(userCred mcclient.TokenCredential) error {
	if userCred == nil || !userCred.HasSystemAdminPrivilege() {
		return httperrors.NewForbiddenError("enabling ansible playbook requires system admin privilege")
	}
	return nil
}

// applyPlaybookEnabledByCred sets enabled on create/update: system admin
// defaults to enabled, others are always disabled.
func applyPlaybookEnabledByCred(userCred mcclient.TokenCredential, data *jsonutils.JSONDict) {
	if userCred != nil && userCred.HasSystemAdminPrivilege() {
		if !data.Contains("enabled") {
			data.Set("enabled", jsonutils.JSONTrue)
		}
		return
	}
	data.Set("enabled", jsonutils.JSONFalse)
}

func ensurePlaybookEnabled(enabled bool) error {
	if !enabled {
		return httperrors.NewForbiddenError("playbook is not enabled")
	}
	return nil
}

func (man *SAnsiblePlaybookManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.AnsiblePlaybookListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	return man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
}

func (man *SAnsiblePlaybookManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	pbV := NewAnsiblePlaybookValidator("playbook", userCred)
	if err := pbV.Validate(ctx, data); err != nil {
		return nil, err
	}
	data.Set("status", jsonutils.NewString(api.AnsiblePlaybookStatusInit))
	var err error
	input := apis.VirtualResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	applyPlaybookEnabledByCred(userCred, data)
	return data, nil
}

func (apb *SAnsiblePlaybook) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	apb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if !apb.GetEnabled() {
		return
	}
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		log.Errorf("postCreate: runPlaybook: %v", err)
	}
}

func (man *SAnsiblePlaybookManager) InitializeData() error {
	pbs := []SAnsiblePlaybook{}
	q := AnsiblePlaybookManager.Query()
	q = q.Filter(sqlchemy.Equals(q.Field("status"), api.AnsiblePlaybookStatusRunning))
	if err := db.FetchModelObjects(AnsiblePlaybookManager, q, &pbs); err != nil {
		return errors.Wrap(err, "fetch running playbooks")
	}
	for i := 0; i < len(pbs); i++ {
		pb := &pbs[i]
		_, err := db.Update(pb, func() error {
			pb.Status = api.AnsiblePlaybookStatusUnknown
			return nil
		})
		if err != nil {
			log.Errorf("set playbook %s(%s) to unknown state: %v", pb.Name, pb.Id, err)
		}
	}
	if err := man.eanbleExistingPlaybooks(); err != nil {
		return errors.Wrap(err, "enable existing playbooks")
	}
	return nil
}

func (man *SAnsiblePlaybookManager) eanbleExistingPlaybooks() error {
	pbs := []SAnsiblePlaybookV2{}
	q := AnsiblePlaybookV2Manager.Query().IsNull("enabled")
	if err := db.FetchModelObjects(AnsiblePlaybookV2Manager, q, &pbs); err != nil {
		return errors.Wrap(err, "fetch running playbooks")
	}
	for i := 0; i < len(pbs); i++ {
		pb := &pbs[i]
		_, err := db.Update(pb, func() error {
			pb.Enabled = tristate.True
			return nil
		})
		if err != nil {
			log.Errorf("enable playbook %s(%s): %v", pb.Name, pb.Id, err)
		}
	}
	return nil
}

func (apb *SAnsiblePlaybook) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if apb.Status == api.AnsiblePlaybookStatusRunning {
		return httperrors.NewConflictError("playbook is in running state")
	}
	return nil
}

func (apb *SAnsiblePlaybook) ValidateUpdateCondition(ctx context.Context) error {
	if apb.Status == api.AnsiblePlaybookStatusRunning {
		return httperrors.NewConflictError("playbook is in running state")
	}
	return nil
}

func (apb *SAnsiblePlaybook) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	pbV := NewAnsiblePlaybookValidator("playbook", userCred)
	if err := pbV.Validate(ctx, data); err != nil {
		return nil, err
	}
	apb.Playbook = pbV.Playbook // Update as a whole
	data.Set("status", jsonutils.NewString(api.AnsiblePlaybookStatusInit))
	applyPlaybookEnabledByCred(userCred, data)
	if enabled, err := data.Bool("enabled"); err == nil {
		apb.SetEnabled(enabled)
	}
	return data, nil
}

func (apb *SAnsiblePlaybook) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	apb.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	if !apb.GetEnabled() {
		return
	}
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		log.Errorf("postUpdate: runPlaybook: %v", err)
	}
}

func (apb *SAnsiblePlaybook) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if err := requireSystemAdmin(userCred); err != nil {
		return nil, err
	}
	if err := ansible.ValidatePlaybook(apb.Playbook); err != nil {
		return nil, httperrors.NewInputParameterError("%s", err.Error())
	}
	if err := db.EnabledPerformEnable(apb, ctx, userCred, true); err != nil {
		return nil, err
	}
	return nil, nil
}

func (apb *SAnsiblePlaybook) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(apb, ctx, userCred, false); err != nil {
		return nil, err
	}
	return nil, nil
}

func (apb *SAnsiblePlaybook) PerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		return nil, httperrors.NewConflictError("%s", err.Error())
	}
	return nil, nil
}

func (apb *SAnsiblePlaybook) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := apb.stopPlaybook(ctx, userCred)
	if err != nil {
		return nil, httperrors.NewConflictError("%s", err.Error())
	}
	return nil, nil
}

func (apb *SAnsiblePlaybook) runPlaybook(ctx context.Context, userCred mcclient.TokenCredential) error {
	man := AnsiblePlaybookManager
	man.sessionsMux.Lock()
	defer man.sessionsMux.Unlock()
	if man.sessions.Has(apb.Id) {
		return fmt.Errorf("playbook is already running")
	}
	if err := ensurePlaybookEnabled(apb.GetEnabled()); err != nil {
		return err
	}

	// init private key
	pb := apb.Playbook.Copy()
	if err := ansible.ValidatePlaybook(pb); err != nil {
		return err
	}
	if len(pb.PrivateKey) == 0 {
		if k, err := compute.Sshkeypairs.FetchPrivateKey(ctx, userCred); err != nil {
			return err
		} else {
			pb.PrivateKey = []byte(k)
		}
	}
	// init tmpdir clean policy
	if options.Options.KeepTmpdir {
		pb.CleanOnExit(false)
	}
	pb.OutputWriter(&ansiblePlaybookOutputWriter{apb})

	_, err := db.Update(apb, func() error {
		apb.StartTime = time.Now()
		apb.EndTime = time.Time{}
		apb.Output = ""
		apb.Status = api.AnsiblePlaybookStatusRunning
		return nil
	})
	if err != nil {
		log.Errorf("run playbook: update db failed before run: %v", err)
	}

	man.sessions.Add(apb.Id, pb)

	go func() {
		defer func() {
			man.sessionsMux.Lock()
			defer man.sessionsMux.Unlock()
			man.sessions.Remove(apb.Id)
		}()
		runErr := man.sessions.Run(apb.Id)

		_, err := db.Update(apb, func() error {
			err := man.sessions.Err(apb.Id)
			if err != nil {
				apb.Status = api.AnsiblePlaybookStatusCanceled
			} else if runErr != nil {
				log.Warningf("playbook %s(%s) failed: %v", apb.Name, apb.Id, runErr)
				apb.Status = api.AnsiblePlaybookStatusFailed
			} else {
				apb.Status = api.AnsiblePlaybookStatusSucceeded
			}
			apb.EndTime = time.Now()
			return nil
		})
		if err != nil {
			log.Errorf("updating ansible playbook failed: %v", err)
		}
	}()
	return nil
}

func (apb *SAnsiblePlaybook) stopPlaybook(ctx context.Context, userCred mcclient.TokenCredential) error {
	man := AnsiblePlaybookManager
	man.sessionsMux.Lock()
	defer man.sessionsMux.Unlock()
	if !man.sessions.Has(apb.Id) {
		if apb.Status == api.AnsiblePlaybookStatusRunning {
			_, err := db.Update(apb, func() error {
				apb.Status = api.AnsiblePlaybookStatusUnknown
				return nil
			})
			if err != nil {
				log.Errorf("updating ansible playbook status to unknown failed: %v", err)
			}
		}
		return fmt.Errorf("playbook is not running")
	}
	// the playbook will be removed from session map in runPlaybook() on return from run
	man.sessions.Stop(apb.Id)
	return nil
}

func (apb *SAnsiblePlaybook) getMaxOutputLength() int {
	return OutputMaxBytes
}

func (apb *SAnsiblePlaybook) getOutput() string {
	return apb.Output
}

func (apb *SAnsiblePlaybook) setOutput(s string) {
	apb.Output = s
}
