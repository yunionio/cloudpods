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

package stringutils2

import "testing"

func TestHashIdsMac(t *testing.T) {
	mac := HashIdsMac("123", "456")
	want := "ff:96:9e:ef:6e:ca"
	if want != mac {
		t.Errorf("got: %s want: %s", mac, want)
	}
}
