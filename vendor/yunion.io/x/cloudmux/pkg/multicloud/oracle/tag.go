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

package oracle

import (
	"github.com/pkg/errors"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SOracleTag struct {
	Tags    map[string]string `json:"freeformTags"`
	SysTags struct {
		OracleTags map[string]string `json:"Oracle-Tags"`
	} `json:"definedTags"`
}

func (ot SOracleTag) GetSysTags() map[string]string {
	return ot.SysTags.OracleTags
}

func (ot SOracleTag) GetTags() (map[string]string, error) {
	return ot.Tags, nil
}

func (ot SOracleTag) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
