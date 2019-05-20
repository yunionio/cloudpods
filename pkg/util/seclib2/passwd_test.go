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

package seclib2

import "testing"

func TestGeneratePassword(t *testing.T) {
	passwd := "Hello world!"
	dk, err := GeneratePassword(passwd)
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	t.Logf("%s", dk)

	err = VerifyPassword(passwd, dk)
	if err != nil {
		t.Errorf("fail to verify %s", err)
	}
}

func TestGeneratePassword2(t *testing.T) {
	passwd := "Hello world!"
	dk, err := BcryptPassword(passwd)
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	t.Logf("%s", dk)

	err = BcryptVerifyPassword(passwd, dk)
	if err != nil {
		t.Errorf("fail to verify %s", err)
	}

	hash := "$2b$12$PhhOkNNNa2wWU643XKVC3uS6cVR8JY4ZkJ2p.GlmZWCiv7oqp2a9m"
	pass := "MxqhTC2VKe067jtD"

	err = BcryptVerifyPassword(pass, hash)
	if err != nil {
		t.Errorf("Verify existing fail %s", err)
	}
}
