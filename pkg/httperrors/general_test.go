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

package httperrors

import (
	"fmt"
	"strings"
	"testing"
)

func TestGeneralError(t *testing.T) {
	t.Run("unclassified", func(t *testing.T) {
		t.Run("no fmt", func(t *testing.T) {
			err := fmt.Errorf("i am an unclassified error")
			jce := NewGeneralError(err)
			if strings.Contains(jce.Details, "%!(EXTRA") {
				t.Errorf("bad error formating: %v", jce)
			}
		})
		t.Run("fmt", func(t *testing.T) {
			err := fmt.Errorf("i am error with plain %%s")
			jce := NewGeneralError(err)
			if strings.Contains(jce.Details, "%!(EXTRA") {
				t.Errorf("bad error formating: %v", jce)
			}
		})
	})
}
