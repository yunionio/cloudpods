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

package pinyinutils

import (
	"strings"

	"github.com/mozillazg/go-pinyin"
)

func Text2Pinyin(hans string) string {
	args := pinyin.NewArgs()
	out := strings.Builder{}
	for _, runeVal := range hans {
		if runeVal > 0x7f {
			o := pinyin.SinglePinyin(runeVal, args)
			for i := range o {
				out.WriteString(o[i])
			}
		} else {
			out.WriteRune(runeVal)
		}
	}
	return out.String()
}
