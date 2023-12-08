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

package monitor

import (
	"crypto/sha256"
	"fmt"

	"yunion.io/x/jsonutils"
)

type MetricQueryInput struct {
	From            string        `json:"from"`
	To              string        `json:"to"`
	Scope           string        `json:"scope"`
	Slimit          string        `json:"slimit"`
	Soffset         string        `json:"soffset"`
	Unit            bool          `json:"unit"`
	Interval        string        `json:"interval"`
	DomainId        string        `json:"domain_id"`
	ProjectId       string        `json:"project_id"`
	MetricQuery     []*AlertQuery `json:"metric_query"`
	Signature       string        `json:"signature"`
	ShowMeta        bool          `json:"show_meta"`
	SkipCheckSeries bool          `json:"skip_check_series"`
}

const (
	QUERY_SIGNATURE_KEY = "signature"
)

func DigestQuerySignature(data *jsonutils.JSONDict) string {
	data.Remove(QUERY_SIGNATURE_KEY)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data.String())))
}
