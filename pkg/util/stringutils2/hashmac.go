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

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func HashIdsMac(ids ...string) string {
	h := sha256.New()
	for _, id := range ids {
		h.Write([]byte(id))
	}
	sum := h.Sum(nil)
	hexStr := make([]string, 6)
	hexStr[0] = "ff"
	for i := 1; i < 6; i++ {
		hexStr[i] = fmt.Sprintf("%02x", sum[i])
	}
	return strings.Join(hexStr, ":")
}
