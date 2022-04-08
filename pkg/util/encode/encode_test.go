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

package encode

import (
	"testing"
)

func TestLabelEncode(t *testing.T) {
	cases := []struct {
		label       string
		encodeLabel string
	}{
		{
			label:       "app/test/hellow",
			encodeLabel: "app_2ftest_2fhellow",
		},
		{
			label:       "app/test_hellow",
			encodeLabel: "app_2ftest_5fhellow",
		},
		{
			label:       "projName",
			encodeLabel: "proj_4eame",
		},
		{
			label:       "你好test",
			encodeLabel: "你好test",
		},
		{
			label:       "你好test:",
			encodeLabel: "你好test_3a",
		},
	}
	for _, label := range cases {
		encodeLabel := EncodeGoogleLabel(label.label)
		if label.encodeLabel != encodeLabel {
			t.Fatalf("encode %s to %s want: %s", label.label, encodeLabel, label.encodeLabel)
		}
		decodeLabel := DecodeGoogleLable(encodeLabel)
		if label.label != decodeLabel {
			t.Fatalf("decode %s to %s want: %s", encodeLabel, decodeLabel, label.label)
		}
	}
}
