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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ansiblev2"
	"yunion.io/x/onecloud/pkg/util/rand"
)

func (router *SRouter) realize(ctx context.Context, userCred mcclient.TokenCredential) error {
	plays := []*ansiblev2.Play{
		router.playEssential(),
	}

	host, err := router.ansibleHost()
	if err != nil {
		return err
	}
	if router.RealizeWgIfaces {
		plays = append(plays,
			router.playInstallWireguard(),
			router.playDeployWireguardNetworks(),
		)
	}

	if router.RealizeRoutes {
		playRoutes, err := router.playDeployRoutes()
		if err != nil {
			return err
		}
		plays = append(plays, playRoutes)
	}
	if router.RealizeRules {
		playRules, err := router.playDeployRules()
		if err != nil {
			return err
		}
		plays = append(plays, playRules)
	}

	inv := ansiblev2.NewInventory()
	inv.SetHost(router.Name, host)
	pb := ansiblev2.NewPlaybook(plays...)
	files := router.playFilesStr()

	params := jsonutils.NewDict()
	params.Set("creator_mark", jsonutils.NewString("router:"+router.Id))
	params.Set("name", jsonutils.NewString(router.Name+"-"+fmt.Sprintf("%d-", router.UpdateVersion)+rand.String(5)))
	params.Set("inventory", jsonutils.NewString(inv.String()))
	params.Set("playbook", jsonutils.NewString(pb.String()))
	params.Set("files", jsonutils.NewString(files))
	cliSess := auth.GetSession(ctx, userCred, "", "")
	if _, err := mcclient_modules.AnsiblePlaybooksV2.Create(cliSess, params); err != nil {
		return errors.WithMessagef(err, "create ansible task")
	}
	return nil
}

func (router *SRouter) AllowPerformRealize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, router, "realize")
}

func (router *SRouter) PerformRealize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := router.realize(ctx, userCred)
	if err != nil {
		return nil, httperrors.NewBadRequestError("%s", err)
	}
	return nil, nil
}
