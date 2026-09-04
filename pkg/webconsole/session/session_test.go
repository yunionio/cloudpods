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

package session

import (
	"context"
	"os/exec"
	"testing"

	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/webconsole/recorder"
)

type testSessionData struct {
	id string
}

func (s *testSessionData) GetId() string {
	return s.id
}

func (s *testSessionData) IsNeedLogin() (bool, error) {
	return false, nil
}

func (s *testSessionData) GetDisplayInfo(ctx context.Context) (*SDisplayInfo, error) {
	return nil, nil
}

func (s *testSessionData) GetProtocol() string {
	return "tty"
}

func (s *testSessionData) GetCommand() *exec.Cmd {
	return nil
}

func (s *testSessionData) GetSafeCommandString() string {
	return ""
}

func (s *testSessionData) Cleanup() error {
	return nil
}

func (s *testSessionData) Scan(byte, func(string)) {}

func (s *testSessionData) GetClientSession() *mcclient.ClientSession {
	return nil
}

func (s *testSessionData) GetRecordObject() *recorder.Object {
	return nil
}

// sessions must be isolated by their unique ids: a token issued for one
// session can not be used to reach another session
func TestSessionIsolation(t *testing.T) {
	man := NewSessionManager()
	id1 := stringutils.UUID4()
	id2 := stringutils.UUID4()
	s1, err := man.Save(&testSessionData{id: id1})
	if err != nil {
		t.Fatalf("save session1: %v", err)
	}
	s2, err := man.Save(&testSessionData{id: id2})
	if err != nil {
		t.Fatalf("save session2: %v", err)
	}
	got, ok := man.Get(s1.AccessToken)
	if !ok {
		t.Fatalf("get session1 by its own token failed")
	}
	if got.Id != id1 {
		t.Fatalf("token of session1 resolved to session %s", got.Id)
	}
	got, ok = man.Get(s2.AccessToken)
	if !ok {
		t.Fatalf("get session2 by its own token failed")
	}
	if got.Id != id2 {
		t.Fatalf("token of session2 resolved to session %s", got.Id)
	}
}

// a token that decrypts to the same session id but is not the issued token
// (e.g. re-encrypted with a recovered key) must be rejected
func TestSessionTokenBinding(t *testing.T) {
	man := NewSessionManager()
	id := stringutils.UUID4()
	s, err := man.Save(&testSessionData{id: id})
	if err != nil {
		t.Fatalf("save session: %v", err)
	}
	if _, ok := man.Get(s.AccessToken); !ok {
		t.Fatalf("get session by its own token failed")
	}
	// forge another ciphertext of the same id
	forged, err := utils.EncryptAESBase64Url(AES_KEY, id)
	if err != nil {
		t.Fatalf("encrypt forged token: %v", err)
	}
	if forged == s.AccessToken {
		t.Fatalf("forged token equals issued token, test broken")
	}
	if _, ok := man.Get(forged); ok {
		t.Fatalf("forged token accepted")
	}
	// random garbage must be rejected as well
	if _, ok := man.Get("not-a-valid-token"); ok {
		t.Fatalf("invalid token accepted")
	}
}

// sessions with an empty id (legacy RDP behavior) must not shadow each
// other: the second save replaces the first and the first token dies
func TestSessionEmptyIdReplacement(t *testing.T) {
	man := NewSessionManager()
	s1, err := man.Save(&testSessionData{id: ""})
	if err != nil {
		t.Fatalf("save session1: %v", err)
	}
	s2, err := man.Save(&testSessionData{id: ""})
	if err != nil {
		t.Fatalf("save session2: %v", err)
	}
	// the old token must no longer resolve to the new session
	if _, ok := man.Get(s1.AccessToken); ok {
		t.Fatalf("replaced session token still accepted")
	}
	got, ok := man.Get(s2.AccessToken)
	if !ok {
		t.Fatalf("latest session token rejected")
	}
	if got != s2 {
		t.Fatalf("token resolved to wrong session")
	}
}
