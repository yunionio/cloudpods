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
	"reflect"
	"testing"

	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestClouduserSecretFieldTags(t *testing.T) {
	ft, ok := reflect.TypeOf(SClouduser{}).FieldByName("Secret")
	if !ok {
		t.Fatal("Secret field missing")
	}
	if got := ft.Tag.Get("list"); got != "" {
		t.Fatalf("Secret list tag = %q, want empty", got)
	}
	if got := ft.Tag.Get("get"); got != "" {
		t.Fatalf("Secret get tag = %q, want empty", got)
	}
}

func TestGetPassword(t *testing.T) {
	plain := "P@ssw0rd12"
	user := &SClouduser{}
	user.Id = "clouduser-id"
	sec, err := utils.EncryptAESBase64(user.Id, plain)
	if err != nil {
		t.Fatalf("EncryptAESBase64: %v", err)
	}
	user.Secret = sec
	got, err := user.GetPassword()
	if err != nil {
		t.Fatalf("GetPassword: %v", err)
	}
	if got != plain {
		t.Fatalf("GetPassword = %q, want %q", got, plain)
	}
}

func TestFormatClouduserLoginInfo(t *testing.T) {
	info := formatClouduserLoginInfo("alice", computeapi.CLOUD_PROVIDER_ALIYUN, "https://signin.aliyun.com/mycorp.onaliyun.com/login.htm", "secret")
	if info.Username != "alice@mycorp.onaliyun.com" {
		t.Fatalf("aliyun username = %q", info.Username)
	}
	if info.Account != "mycorp.onaliyun.com" {
		t.Fatalf("aliyun account = %q", info.Account)
	}
	if info.Password != "secret" || info.Url == "" {
		t.Fatalf("unexpected login info: %+v", info)
	}

	info = formatClouduserLoginInfo("bob", computeapi.CLOUD_PROVIDER_AWS, "https://123456789012.signin.aws.amazon.com/console", "secret")
	if info.Account != "123456789012" {
		t.Fatalf("aws account = %q", info.Account)
	}
	if info.Username != "bob" {
		t.Fatalf("aws username = %q", info.Username)
	}

	info = formatClouduserLoginInfo("carol", computeapi.CLOUD_PROVIDER_QCLOUD, "https://cloud.tencent.com/login?account=corp", "secret")
	if info.Account != "corp" {
		t.Fatalf("qcloud account = %q", info.Account)
	}
}
