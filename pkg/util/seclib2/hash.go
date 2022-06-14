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

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

func HashId(seed string, idx byte, width int) string {
	h := sha256.New()
	h.Write([]byte(seed))
	if idx > 0 {
		h.Write([]byte{idx})
	}
	sum := h.Sum(nil)
	numStr := fmt.Sprintf(fmt.Sprintf("%%0%dx", width), binary.BigEndian.Uint64(sum[:8]))
	if len(numStr) > width {
		numStr = numStr[:width]
	}
	return numStr
}
