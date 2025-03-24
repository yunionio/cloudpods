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

package cloudprovider

import (
	"time"

	"yunion.io/x/jsonutils"
)

type SClouduserCreateConfig struct {
	Name                  string
	Desc                  string
	Password              string
	IsConsoleLogin        bool
	Email                 string
	MobilePhone           string
	UserType              string
	EnableMfa             bool
	PasswordResetRequired bool
}

type SCloudpolicyPermission struct {
	Name     string
	Action   string
	Category string
}

type SCloudpolicyCreateOptions struct {
	Name     string
	Desc     string
	Document *jsonutils.JSONDict
}

type SAccessKey struct {
	Name      string
	AccessKey string
	Secret    string
	Status    string
	CreatedAt time.Time
}
