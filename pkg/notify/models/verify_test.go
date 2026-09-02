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
	"testing"
)

func TestGenerateVerifyToken(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		token, err := VerificationManager.generateVerifyToken()
		if err != nil {
			t.Fatalf("generateVerifyToken: %v", err)
		}
		if len(token) != 6 {
			t.Fatalf("token length %d, want 6", len(token))
		}
		for _, c := range token {
			if c < '0' || c > '9' {
				t.Fatalf("token %q contains non-digit", token)
			}
		}
		if seen[token] {
			t.Fatalf("duplicate token %q generated within 200 draws", token)
		}
		seen[token] = true
	}
}
