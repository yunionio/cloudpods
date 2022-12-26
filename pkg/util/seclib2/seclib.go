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
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

func RandomPassword2(width int) string {
	return seclib.RandomPassword2(width)
}

func AnalyzePasswordStrenth(passwd string) seclib.PasswordStrength {
	return seclib.AnalyzePasswordStrenth(passwd)
}

func ValidatePassword(passwd string) error {
	ps := AnalyzePasswordStrenth(passwd)
	if len(ps.Invalid) > 0 {
		return httperrors.NewInputParameterError("invalid characters %s", string(ps.Invalid))
	}
	if utils.IsInStringArray(passwd, seclib.WEAK_PASSWORDS) || !ps.MeetComplexity() {
		return httperrors.NewWeakPasswordError()
	}
	return nil
}

func MeetComplxity(passwd string) bool {
	return seclib.MeetComplxity(passwd)
}
