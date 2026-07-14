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

package codexconfig

import "testing"

func TestBuildCatalogVisualModalities(t *testing.T) {
	catalog := BuildCatalogFromIDs([]ModelListEntry{{
		ID:              "deepseek-v4-pro",
		OwnedBy:         "deepseek",
		InputModalities: []string{"text", "image"},
	}})
	if len(catalog) != 1 {
		t.Fatalf("catalog len = %d", len(catalog))
	}
	if len(catalog[0].InputModalities) != 2 || catalog[0].InputModalities[1] != "image" {
		t.Fatalf("modalities = %#v", catalog[0].InputModalities)
	}
}
