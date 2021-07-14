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
	"crypto/sha256"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	QUERY_SIGNATURE_KEY = "signature"
)

func digestQuerySignature(data *jsonutils.JSONDict) string {
	data.Remove(QUERY_SIGNATURE_KEY)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data.String())))
}

func ValidateQuerySignature(input jsonutils.JSONObject) error {
	data, ok := input.(*jsonutils.JSONDict)
	if !ok {
		return httperrors.NewInputParameterError("input not json dict")
	}
	signature, err := data.GetString(QUERY_SIGNATURE_KEY)
	if err != nil {
		if errors.Cause(err) == jsonutils.ErrJsonDictKeyNotFound {
			return httperrors.NewNotFoundError("not found signature")
		}
		return errors.Wrap(err, "get signature")
	}
	if signature != digestQuerySignature(data) {

		return httperrors.NewBadRequestError("signature error")
	}
	return nil
}
