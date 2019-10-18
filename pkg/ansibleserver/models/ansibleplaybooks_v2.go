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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/ansiblev2"
)

// This is at the moment for internal use only.  Update is not allowed
type SAnsiblePlaybookV2 struct {
	db.SVirtualResourceBase

	Playbook  string    `length:"text" nullable:"false" create:"required" get:"user"`
	Inventory string    `length:"text" nullable:"false" create:"required" get:"user"`
	Files     string    `length:"text" nullable:"false" create:"optional" get:"user"`
	Output    string    `length:"medium" get:"user"`
	StartTime time.Time `list:"user"`
	EndTime   time.Time `list:"user"`

	CreatorMark string `length:"32" nullable:"false" create:"optional" get:"user"`
}

type SAnsiblePlaybookV2Manager struct {
	db.SVirtualResourceBaseManager

	sessions    ansible.SessionManager
	sessionsMux *sync.Mutex
}

var AnsiblePlaybookV2Manager *SAnsiblePlaybookV2Manager

func init() {
	AnsiblePlaybookV2Manager = &SAnsiblePlaybookV2Manager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAnsiblePlaybookV2{},
			"ansibleplaybooks_v2_tbl",
			"ansibleplaybook_v2",
			"ansibleplaybooks_v2",
		),
		sessions:    ansible.SessionManager{},
		sessionsMux: &sync.Mutex{},
	}
	AnsiblePlaybookV2Manager.SetVirtualObject(AnsiblePlaybookV2Manager)
}

func (man *SAnsiblePlaybookV2Manager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data.Set("status", jsonutils.NewString(AnsiblePlaybookStatusInit))
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (apb *SAnsiblePlaybookV2) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	apb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		log.Errorf("postCreate: runPlaybook: %v", err)
	}
}

func (man *SAnsiblePlaybookV2Manager) InitializeData() error {
	pbs := []SAnsiblePlaybookV2{}
	q := AnsiblePlaybookV2Manager.Query()
	q = q.Filter(sqlchemy.Equals(q.Field("status"), AnsiblePlaybookStatusRunning))
	if err := db.FetchModelObjects(AnsiblePlaybookV2Manager, q, &pbs); err != nil {
		return errors.WithMessage(err, "fetch running playbooks")
	}
	for i := 0; i < len(pbs); i++ {
		pb := &pbs[i]
		_, err := db.Update(pb, func() error {
			pb.Status = AnsiblePlaybookStatusUnknown
			return nil
		})
		if err != nil {
			log.Errorf("set playbook %s(%s) to unknown state: %v", pb.Name, pb.Id, err)
		}
	}
	return nil
}

func (apb *SAnsiblePlaybookV2) ValidateDeleteCondition(ctx context.Context) error {
	if apb.Status == AnsiblePlaybookStatusRunning {
		return httperrors.NewConflictError("playbook is in running state")
	}
	return nil
}

func (apb *SAnsiblePlaybookV2) AllowPerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return apb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, apb, "run")
}

func (apb *SAnsiblePlaybookV2) PerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		return nil, httperrors.NewConflictError("%s", err.Error())
	}
	return nil, nil
}

func (apb *SAnsiblePlaybookV2) AllowPerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return apb.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, apb, "stop")
}

func (apb *SAnsiblePlaybookV2) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := apb.stopPlaybook(ctx, userCred)
	if err != nil {
		return nil, httperrors.NewConflictError("%s", err.Error())
	}
	return nil, nil
}

func (apb *SAnsiblePlaybookV2) runPlaybook(ctx context.Context, userCred mcclient.TokenCredential) error {
	man := AnsiblePlaybookV2Manager
	man.sessionsMux.Lock()
	defer man.sessionsMux.Unlock()
	if man.sessions.Has(apb.Id) {
		return fmt.Errorf("playbook is already running")
	}

	var (
		privateKey string
		err        error
		files      = map[string][]byte{}
	)
	if apb.Files != "" {
		obj, err := jsonutils.ParseString(apb.Files)
		if err != nil {
			return fmt.Errorf("playbook files json: parse: %v", err)
		}
		filesJ, err := obj.GetMap()
		if err != nil {
			return fmt.Errorf("playbook files json: get map: %v", err)
		}
		for name, obj := range filesJ {
			content, err := obj.GetString()
			if err != nil {
				return fmt.Errorf("playbook files json: get content %s: %v", name, err)
			}
			files[name] = []byte(content)
		}
	}
	// init private key
	if privateKey, err = fetchPrivateKey(ctx, userCred); err != nil {
		return err
	}

	_, err = db.Update(apb, func() error {
		apb.StartTime = time.Now()
		apb.EndTime = time.Time{}
		apb.Output = ""
		apb.Status = AnsiblePlaybookStatusRunning
		return nil
	})
	if err != nil {
		log.Errorf("run playbook: update db failed before run: %v", err)
	}

	sess := ansiblev2.NewSession().
		Inventory(apb.Inventory).
		Playbook(apb.Playbook).
		PrivateKey(privateKey).
		Files(files).
		OutputWriter(&ansiblePlaybookOutputWriter{apb})
	man.sessions.Add(apb.Id, sess)

	// NOTE host state check? run only on online hosts and running guests, skip others
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
				apb.Status = AnsiblePlaybookStatusCanceled
			} else if runErr != nil {
				log.Warningf("playbook %s(%s) failed: %v", apb.Name, apb.Id, runErr)
				apb.Status = AnsiblePlaybookStatusFailed
			} else {
				apb.Status = AnsiblePlaybookStatusSucceeded
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

func (apb *SAnsiblePlaybookV2) stopPlaybook(ctx context.Context, userCred mcclient.TokenCredential) error {
	man := AnsiblePlaybookV2Manager
	man.sessionsMux.Lock()
	defer man.sessionsMux.Unlock()
	if !man.sessions.Has(apb.Id) {
		return fmt.Errorf("playbook is not running")
	}
	// the playbook will be removed from session map in runPlaybook() on return from run
	man.sessions.Stop(apb.Id)
	return nil
}

func (apb *SAnsiblePlaybookV2) getMaxOutputLength() int {
	return OutputMaxBytes
}

func (apb *SAnsiblePlaybookV2) getOutput() string {
	return apb.Output
}

func (apb *SAnsiblePlaybookV2) setOutput(s string) {
	apb.Output = s
}
