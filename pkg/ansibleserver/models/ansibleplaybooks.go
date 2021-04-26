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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/ansibleserver/options"
	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

// low priority
//
// - retry times and interval
// - timeout,
// - copied playbook has no added value

type SAnsiblePlaybook struct {
	db.SVirtualResourceBase

	Playbook  *ansible.Playbook `length:"text" nullable:"false" create:"required" get:"user" update:"user"`
	Output    string            `length:"medium" get:"user"`
	StartTime time.Time         `list:"user"`
	EndTime   time.Time         `list:"user"`
}

type SAnsiblePlaybookManager struct {
	db.SVirtualResourceBaseManager

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

func (man *SAnsiblePlaybookManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	pbV := NewAnsiblePlaybookValidator("playbook", userCred)
	if err := pbV.Validate(data); err != nil {
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
	return data, nil
}

func (apb *SAnsiblePlaybook) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	apb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
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
	return nil
}

func (apb *SAnsiblePlaybook) ValidateDeleteCondition(ctx context.Context) error {
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
	if err := pbV.Validate(data); err != nil {
		return nil, err
	}
	apb.Playbook = pbV.Playbook // Update as a whole
	data.Set("status", jsonutils.NewString(api.AnsiblePlaybookStatusInit))
	return data, nil
}

func (apb *SAnsiblePlaybook) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	apb.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		log.Errorf("postUpdate: runPlaybook: %v", err)
	}
}

func (apb *SAnsiblePlaybook) AllowPerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return apb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, apb, "run")
}

func (apb *SAnsiblePlaybook) PerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		return nil, httperrors.NewConflictError("%s", err.Error())
	}
	return nil, nil
}

func (apb *SAnsiblePlaybook) AllowPerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return apb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, apb, "stop")
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

	// init private key
	pb := apb.Playbook.Copy()
	if len(pb.PrivateKey) == 0 {
		if k, err := mcclient_modules.Sshkeypairs.FetchPrivateKey(ctx, userCred); err != nil {
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
