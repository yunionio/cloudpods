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

package vpcagent

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/vpcagent/models"
)

type Result struct {
	Models  *models.ModelSets
	Correct bool
	Changed bool
}

func GetTopoResult(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (interface{}, error) {
	mss := models.NewModelSets()
	s := auth.GetAdminSession(ctx, "")
	r, err := apihelper.SyncDBModelSets(mss, s, &apihelper.Options{
		ListBatchSize:        1024,
		IncludeDetails:       false,
		IncludeOtherCloudEnv: false,
	})
	ret := &Result{
		Models:  mss,
		Correct: r.Correct,
		Changed: r.Changed,
	}
	return ret, err
}
