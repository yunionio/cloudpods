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

package utils

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	session *mcclient.ClientSession
)

func InitSession(options *options.CommonOptions) {
	session = auth.GetAdminSession(context.Background(), options.Region, "v3")
}

func GetUserByID(id string) (jsonutils.JSONObject, error) {
	return modules.UsersV3.Get(session, id, jsonutils.NewDict())
}

func GetUsersByGroupID(gid string) ([]string, error) {
	ret, err := modules.Groups.GetUsers(session, gid)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(ret.Data))
	for i := range ret.Data {
		ids[i], _ = ret.Data[i].GetString("id")
	}
	return ids, nil
}

func GetUsernameByID(id string) (string, error) {
	user, err := GetUserByID(id)
	if err != nil {
		return "", err
	}
	name, _ := user.GetString("name")
	return name, nil
}
