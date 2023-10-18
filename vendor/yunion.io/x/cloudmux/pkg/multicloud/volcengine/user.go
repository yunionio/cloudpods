// Copyright 2023 Yunion
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

package volcengine

import (
	"time"

	"yunion.io/x/pkg/errors"
)

type SUser struct {
	Id                  int
	CreateDate          time.Time
	UpdateDate          time.Time
	Status              string
	AccountId           string
	UserName            string
	Description         string
	DisplayName         string
	Email               string
	EmailIsVerify       bool
	MobilePhone         string
	MobilePhoneIsVerify bool
	Trn                 string
	Source              string
}

type SCallerIdentity struct {
	AccountId    string
	UserId       string
	RoleId       string
	PrincipalId  string
	IdentityType string
}

func (client *SVolcEngineClient) GetCallerIdentity() (*SCallerIdentity, error) {
	// sys is not currently supported
	params := map[string]string{}
	body, err := client.iamRequest("", "ListUsers", params)
	if err != nil {
		return nil, err
	}
	id := &SCallerIdentity{}
	users := []SUser{}
	err = body.Unmarshal(&users, "Result", "UserMetadata")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	id.AccountId = users[0].AccountId
	return id, nil
}
