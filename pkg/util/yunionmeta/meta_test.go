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

package yunionmeta

import (
	"context"
	"testing"

	"yunion.io/x/pkg/util/httputils"
)

func TestCurrencyRate(t *testing.T) {
	meta := &SSkuResourcesMeta{}
	metaUrl := "https://yunionmeta.oss-cn-beijing.aliyuncs.com/sku.meta"
	_, body, err := httputils.JSONRequest(nil, context.TODO(), "GET", metaUrl, nil, nil, false)
	err = body.Unmarshal(meta)
	if err != nil {
		t.Fatalf("Unmarshal meta error: %v", err)
	}
	rate, err := meta.GetCurrencyRate("CNY", "USD")
	if err != nil {
		t.Fatalf("get used rate error: %v", rate)
	}
	t.Logf("USD currency is %v", rate)
}
