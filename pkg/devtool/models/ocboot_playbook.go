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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	ansibleapi "yunion.io/x/onecloud/pkg/apis/ansibleserver"
	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
)

const ocbootPlaybookPath = "/opt/yunion/playbook/ocboot-run/playbook.yaml"

// EnsureOcbootPlaybookReferenceLoop creates the ocboot ansible playbook
// reference once ansibleserver is reachable and the runner file exists.
func EnsureOcbootPlaybookReferenceLoop() {
	for {
		err := ensureOcbootPlaybookReference(context.Background())
		if err == nil {
			log.Infof("ocboot ansible playbook reference is ready")
			return
		}
		log.Errorf("ensure ocboot ansible playbook reference: %v, retry in 30s", err)
		time.Sleep(30 * time.Second)
	}
}

func ensureOcbootPlaybookReference(ctx context.Context) error {
	session := auth.GetAdminSession(ctx, "")
	_, err := ansible.AnsiblePlaybookReference.GetByName(session, api.OcbootPlaybookReferenceName, nil)
	if err == nil {
		return nil
	}
	if httputils.ErrorCode(err) != 404 {
		return errors.Wrap(err, "get ocboot ansible playbook reference")
	}
	isPublic := true
	params := ansibleapi.AnsiblePlaybookReferenceCreateInput{}
	params.SAnsiblePlaybookReference.Name = api.OcbootPlaybookReferenceName
	params.SharableVirtualResourceCreateInput.Name = api.OcbootPlaybookReferenceName
	params.Project = "system"
	params.IsPublic = &isPublic
	params.PublicScope = "system"
	params.PlaybookPath = ocbootPlaybookPath
	params.Method = ansibleapi.APReferenceMethodOffline
	_, err = ansible.AnsiblePlaybookReference.Create(session, jsonutils.Marshal(params))
	if err != nil {
		if httputils.ErrorCode(err) == 409 {
			return nil
		}
		return errors.Wrap(err, "create ocboot ansible playbook reference")
	}
	return nil
}
