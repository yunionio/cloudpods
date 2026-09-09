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

package disk

import (
	"testing"
)

func TestJoinCachedImagePath(t *testing.T) {
	base := "/opt/cloud/workspace/image_cache/img1"
	got, err := joinCachedImagePath(base, "/Qwen3.8-27B-NVFP4")
	if err != nil {
		t.Fatalf("abs key: %v", err)
	}
	if got != base+"/Qwen3.8-27B-NVFP4" {
		t.Fatalf("got %q", got)
	}
	got, err = joinCachedImagePath(base, "Qwen3.8-27B-NVFP4")
	if err != nil {
		t.Fatalf("rel key: %v", err)
	}
	if got != base+"/Qwen3.8-27B-NVFP4" {
		t.Fatalf("got %q", got)
	}
	if _, err := joinCachedImagePath(base, "../etc"); err == nil {
		t.Fatal("expected escape to fail")
	}
	if _, err := joinCachedImagePath(base, "/"); err == nil {
		t.Fatal("expected root key to fail")
	}
}
