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
	"strings"
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
	"yunion.io/x/onecloud/pkg/util/ansiblev2"
)

// This is at the moment for internal use only.  Update is not allowed
type SAnsiblePlaybookV2 struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase `enabled->default:"" enabled->nullable:"true"`

	Playbook     string    `length:"text" nullable:"false" create:"required" get:"user"`
	Inventory    string    `length:"text" nullable:"false" create:"required" get:"user"`
	Requirements string    `length:"text" nullable:"false" create:"optional" get:"user"`
	Files        string    `length:"text" nullable:"false" create:"optional" get:"user"`
	Output       string    `length:"medium" get:"user"`
	StartTime    time.Time `list:"user"`
	EndTime      time.Time `list:"user"`

	CreatorMark string `length:"32" nullable:"true" create:"optional" get:"user"`
}

type SAnsiblePlaybookV2Manager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager

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

func (man *SAnsiblePlaybookV2Manager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.AnsiblePlaybookListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	return man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
}

func (man *SAnsiblePlaybookV2Manager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data.Set("status", jsonutils.NewString(api.AnsiblePlaybookStatusInit))
	input := apis.VirtualResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	if err := validateAnsiblePlaybookV2Input(userCred, data); err != nil {
		return nil, err
	}
	applyPlaybookEnabledByCred(userCred, data)
	return data, nil
}

func validateAnsiblePlaybookV2Input(userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	playbook, _ := data.GetString("playbook")
	inventory, _ := data.GetString("inventory")
	files, _ := data.GetString("files")
	requirements, _ := data.GetString("requirements")
	return validateAnsiblePlaybookV2Fields(userCred, playbook, inventory, files, requirements)
}

func validateAnsiblePlaybookV2Fields(userCred mcclient.TokenCredential, playbook, inventory, files, requirements string) error {
	if err := ansible.ValidatePlaybookYAML(playbook); err != nil {
		return httperrors.NewInputParameterError("%s", err.Error())
	}
	if err := ansible.ValidateInventoryYAML(inventory); err != nil {
		return httperrors.NewInputParameterError("%s", err.Error())
	}
	if err := ansible.ValidateFilesJSON(files); err != nil {
		return httperrors.NewInputParameterError("%s", err.Error())
	}
	if strings.TrimSpace(requirements) != "" && (userCred == nil || !userCred.HasSystemAdminPrivilege()) {
		return httperrors.NewForbiddenError("requirements is not allowed")
	}
	return nil
}

func (apb *SAnsiblePlaybookV2) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	apb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if !apb.GetEnabled() {
		return
	}
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		log.Errorf("postCreate: runPlaybook: %v", err)
	}
}

func (man *SAnsiblePlaybookV2Manager) InitializeData() error {
	pbs := []SAnsiblePlaybookV2{}
	q := AnsiblePlaybookV2Manager.Query()
	q = q.Filter(sqlchemy.Equals(q.Field("status"), api.AnsiblePlaybookStatusRunning))
	if err := db.FetchModelObjects(AnsiblePlaybookV2Manager, q, &pbs); err != nil {
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

func (man *SAnsiblePlaybookV2Manager) eanbleExistingPlaybooks() error {
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

func (apb *SAnsiblePlaybookV2) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if apb.Status == api.AnsiblePlaybookStatusRunning {
		return httperrors.NewConflictError("playbook is in running state")
	}
	return nil
}

func (apb *SAnsiblePlaybookV2) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if err := requireSystemAdmin(userCred); err != nil {
		return nil, err
	}
	if err := validateAnsiblePlaybookV2Fields(userCred, apb.Playbook, apb.Inventory, apb.Files, apb.Requirements); err != nil {
		return nil, err
	}
	if err := db.EnabledPerformEnable(apb, ctx, userCred, true); err != nil {
		return nil, err
	}
	return nil, nil
}

func (apb *SAnsiblePlaybookV2) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(apb, ctx, userCred, false); err != nil {
		return nil, err
	}
	return nil, nil
}

func (apb *SAnsiblePlaybookV2) PerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := apb.runPlaybook(ctx, userCred)
	if err != nil {
		return nil, httperrors.NewConflictError("%s", err.Error())
	}
	return nil, nil
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
	if err := ensurePlaybookEnabled(apb.GetEnabled()); err != nil {
		return err
	}
	if err := ansible.ValidatePlaybookYAML(apb.Playbook); err != nil {
		return err
	}
	if err := ansible.ValidateInventoryYAML(apb.Inventory); err != nil {
		return err
	}
	if err := ansible.ValidateFilesJSON(apb.Files); err != nil {
		return err
	}
	if strings.TrimSpace(apb.Requirements) != "" && !userCred.HasSystemAdminPrivilege() {
		return httperrors.NewForbiddenError("requirements is not allowed")
	}

	// hack: force Sleep 50s to wait some host ssh service started
	// time.Sleep(50 * time.Second)

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
	if privateKey, err = compute.Sshkeypairs.FetchPrivateKey(ctx, userCred); err != nil {
		return errors.Wrap(err, "fetch private key")
	}

	_, err = db.Update(apb, func() error {
		apb.StartTime = time.Now()
		apb.EndTime = time.Time{}
		apb.Output = ""
		apb.Status = api.AnsiblePlaybookStatusRunning
		return nil
	})
	if err != nil {
		log.Errorf("run playbook: update db failed before run: %v", err)
		return errors.Wrap(err, "update db failed before run")
	}

	convertGalaxyMirrors := func(requirements string, origin string, mirror string) string {
		return strings.ReplaceAll(requirements, origin, mirror)
	}

	requirements := apb.Requirements
	if len(options.Options.GalaxyMirrors) > 0 {
		for origin, mirror := range options.Options.GalaxyMirrors {
			requirements = convertGalaxyMirrors(requirements, origin, mirror)
		}
	}

	log.Debugf("run playbook: requirements: %s", requirements)

	sess := ansiblev2.NewSession().
		Inventory(apb.Inventory).
		Playbook(apb.Playbook).
		PrivateKey(privateKey).
		Requirements(requirements).
		Files(files).
		OutputWriter(&ansiblePlaybookOutputWriter{apb}).
		KeepTmpdir(options.Options.KeepTmpdir).
		RolePublic(options.Options.RolePublic).
		Timeout(options.Options.Timeout)
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
