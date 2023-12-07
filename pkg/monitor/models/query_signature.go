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

package models

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func ValidateQuerySignature(input jsonutils.JSONObject) error {
	data, ok := input.(*jsonutils.JSONDict)
	if !ok {
		return httperrors.NewInputParameterError("input not json dict")
	}
	signature, err := data.GetString(monitor.QUERY_SIGNATURE_KEY)
	if err != nil {
		if errors.Cause(err) == jsonutils.ErrJsonDictKeyNotFound {
			return httperrors.NewNotFoundError("not found signature")
		}
		return errors.Wrap(err, "get signature")
	}
	if signature != monitor.DigestQuerySignature(data) {
		return httperrors.NewBadRequestError("signature error")
	}
	return nil
}
