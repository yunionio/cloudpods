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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAnsiblePlaybookReference struct {
	db.SSharableVirtualResourceBase

	PlaybookPath  string               `length:"text" nullable:"false" create:"required" get:"user" list:"user"`
	Method        string               `width:"8" nullable:"false" default:"offline" get:"user" list:"user"`
	DefaultParams jsonutils.JSONObject `get:"user" list:"user"`
}

type SAnsiblePlaybookReferenceManager struct {
	db.SSharableVirtualResourceBaseManager
}

var AnsiblePlaybookReferenceManager *SAnsiblePlaybookReferenceManager

func init() {
	AnsiblePlaybookReferenceManager = &SAnsiblePlaybookReferenceManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SAnsiblePlaybookReference{},
			"ansibleplaybook_reference_tbl",
			"ansibleplaybookreference",
			"ansibleplaybookreferences",
		),
	}
	AnsiblePlaybookReferenceManager.SetVirtualObject(AnsiblePlaybookReferenceManager)
}

func (arm *SAnsiblePlaybookReferenceManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

var (
	monitorAgent   = "monitor agent"
	monitorAgentId = "monitoragent"
)

func (arm *SAnsiblePlaybookReferenceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.AnsiblePlaybookReferenceCreateInput) (api.AnsiblePlaybookReferenceCreateInput, error) {

	if input.Method != api.APReferenceMethodOffline {
		return input, httperrors.NewInputParameterError("unkown Method %q", input.Method)
	}
	if !arm.checkOfflinePath(input.PlaybookPath) {
		return input, httperrors.NewInputParameterError("non-existent path: %q", input.PlaybookPath)
	}
	return input, nil
}

func (arm *SAnsiblePlaybookReferenceManager) checkOfflinePath(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func (ar *SAnsiblePlaybookReference) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if data.Contains("playbook_params") {
		params, _ := data.Get("playbook_params")
		ar.DefaultParams = params
	}
	ar.Status = api.APReferenceStatusReady
	return nil
}

func (ar *SAnsiblePlaybookReference) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.AnsiblePlaybookReferenceUpdateInput) (api.AnsiblePlaybookReferenceUpdateInput, error) {
	return input, httperrors.NewForbiddenError("prohibited operation")
}

func (ar *SAnsiblePlaybookReference) PerformRun(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.AnsiblePlaybookReferenceRunInput) (api.AnsiblePlaybookReferenceRunOutput, error) {
	output := api.AnsiblePlaybookReferenceRunOutput{}
	ai, err := AnsiblePlaybookInstanceManager.createInstance(ctx, ar.Id, input.Host, input.Args)
	if err != nil {
		return output, errors.Wrap(err, "unable to create instance")
	}
	output.AnsiblePlaybookInstanceId = ai.Id
	err = ai.runPlaybook(ctx, userCred, ar)
	if err != nil {
		return output, errors.Wrap(err, "unable to runPlaybook")
	}
	return output, nil
}

func (ar *SAnsiblePlaybookReference) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.AnsiblePlaybookReferenceStopInput) (jsonutils.JSONObject, error) {
	obj, err := AnsiblePlaybookInstanceManager.FetchById(input.AnsiblePlaybookInstanceId)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch ansibleplaybookinstance")
	}
	ai := obj.(*SAnsiblePlaybookInstance)
	return nil, ai.stopPlaybook(ctx, userCred)
}
