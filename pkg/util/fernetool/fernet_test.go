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

package fernetool

import (
	"crypto/rand"
	"testing"
)

func TestFernetKeys(t *testing.T) {
	m := SFernetKeyManager{}
	err := m.InitKeys("", 2)
	if err != nil {
		t.Fatalf("fail to initkeys %s", err)
	}
	buf := make([]byte, 128)
	for i := 0; i < 10; i += 1 {
		msgLen, err := rand.Read(buf)
		if err != nil {
			t.Fatalf("rand.Read fail %s", err)
		}
		msg, err := m.Encrypt(buf[:msgLen])
		if err != nil {
			t.Fatalf("fail to encrypt %s", err)
		}
		omsg := m.Decrypt(msg)
		if len(omsg) != msgLen {
			t.Fatalf("descrupt fail %s", err)
		}
		for i := 0; i < msgLen; i += 1 {
			if omsg[i] != buf[i] {
				t.Fatalf("not identical message!!")
			}
		}
	}
}
