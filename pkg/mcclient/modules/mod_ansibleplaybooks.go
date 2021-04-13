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

package modules

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	ansible_apis "yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/mcclient"
	mcclient_models "yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type AnsiblePlaybookManager struct {
	modulebase.ResourceManager
}

var (
	AnsiblePlaybooks AnsiblePlaybookManager
)

func init() {
	AnsiblePlaybooks = AnsiblePlaybookManager{
		NewAnsibleManager(
			"ansibleplaybook",
			"ansibleplaybooks",
			[]string{
				"id",
				"name",
				"status",
				"start_time",
				"end_time",
			},
			[]string{},
		),
	}
	registerV2(&AnsiblePlaybooks)
}

func (man *AnsiblePlaybookManager) UpdateOrCreatePbModel(
	ctx context.Context,
	cliSess *mcclient.ClientSession,
	pbId string,
	pbName string,
	pb *ansible.Playbook,
) (*mcclient_models.AnsiblePlaybook, error) {
	if pbId == "" {
		pbJson, err := man.Get(cliSess, pbName, nil)
		if err == nil {
			pbModel := &mcclient_models.AnsiblePlaybook{}
			if err := pbJson.Unmarshal(pbModel); err == nil {
				pbId = pbModel.Id
			}
		}
	}

	var pbJson jsonutils.JSONObject
	if pbId != "" {
		var err error
		ansiblePbInput := &ansible_apis.AnsiblePlaybookUpdateInput{
			Name:     pbName,
			Playbook: *pb,
		}
		params := ansiblePbInput.JSON(ansiblePbInput)
		pbJson, err = man.Update(cliSess, pbId, params)
		if err != nil {
			return nil, errors.Wrap(err, "update ansibleplaybook")
		}
	} else {
		var err error
		ansiblePbInput := &ansible_apis.AnsiblePlaybookCreateInput{
			Name:     pbName,
			Playbook: *pb,
		}
		params := ansiblePbInput.JSON(ansiblePbInput)
		pbJson, err = man.Create(cliSess, params)
		if err != nil {
			return nil, errors.Wrap(err, "create ansibleplaybook")
		}
	}

	pbModel := &mcclient_models.AnsiblePlaybook{}
	if err := pbJson.Unmarshal(pbModel); err != nil {
		return nil, errors.Wrap(err, "unmarshal ansibleplaybook")
	}
	return pbModel, nil
}
