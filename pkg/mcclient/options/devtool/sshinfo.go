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

package devtool

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SshInfoCreateOptions struct {
	ServerId string `help:"server id"`
}

func (so *SshInfoCreateOptions) Params() (jsonutils.JSONObject, error) {
	body := jsonutils.Marshal(so)
	body.(*jsonutils.JSONDict).Set("generate_name", jsonutils.NewString(so.ServerId))
	return body, nil
}

type SshInfoListOptions struct {
	options.BaseListOptions
}

func (so *SshInfoListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(so)
}

type SshInfoOptions struct {
	ID string `help:"id or name of sshinfo"`
}

func (so *SshInfoOptions) GetId() string {
	return so.ID
}

func (so *SshInfoOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
