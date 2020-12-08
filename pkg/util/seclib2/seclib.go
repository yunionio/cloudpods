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
	"bytes"
	"fmt"
	"math/rand"
	"strings"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	DIGITS  = "23456789"
	LETTERS = "abcdefghjkmnpqrstuvwxyz"
	UPPERS  = "ABCDEFGHJKMNPRSTUVWXYZ"
	PUNC    = "@^-+="

	ALL_DIGITS  = "0123456789"
	ALL_LETTERS = "abcdefghijklmnopqrstuvwxyz"
	ALL_UPPERS  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ALL_PUNC    = "~`!@#$%^&*()-_=+[]{}|:';\",./<>?"
)

type PasswordStrength struct {
	Digits     int
	Lowercases int
	Uppercases int
	Punctuats  int
	Invalid    []byte
}

var WEAK_PASSWORDS []string = []string{
	"Huawei@", "huawei@", "Admin@", "admin@", "Root@", "root@", "ABC@", "abc@", "ABCD@", "abcd@", "Huawei123@", "huawei123@", "Admin123@", "admin123@", "Root123@", "root123@", "Huawei#", "huawei#", "Admin#", "admin#", "Root#", "root#", "ABC#", "abc#", "ABCD#", "abcd#", "Huawei123#", "huawei123#", "Admin123#", "admin123#", "Root123#", "root123#，Huawei!", "huawei!", "Admin!", "admin!", "Root!", "root!", "ABC!", "abc!", "ABCD!", "abcd!", "Huawei123!", "huawei123!", "Admin123!", "admin123!", "Root123!", "root123!", "ABC123!", "abc123!", "Huawei@123", "huawei@123", "Admin@123", "admin@123", "Root@123", "root@123", "ABC@123", "abc@123", "123@Huawei", "123@Root", "123@abc", "Huawei123", "huawei123", "Admin123", "admin123", "Root123", "root123", "abc123", "Huawei_123", "huawei_123", "Admin_123", "admin_123", "Root_123", "root_123", "ABC_123", "abc_123", "123abc", "123abcd", "1234abc", "1234abcd", "abcd123", "abc1234", "abcd1234", "abcd@1234", "abcd1234!", "abcd_1234", "a123456", "123.com", "123@com", "123_com", "Huawei!@#", "huawei!@#", "Admin!@#", "admin!@#", "Root!@#", "root!@#", "Huawei!@", "huawei!@", "Admin!@", "admin!@", "Root!@", "root!@", "Huaweiroot", "HuaweiRoot", "huaweiroot", "huaweiRoot", "Huaweiadmin", "HuaweiAdmin", "huaweiadmin", "huaweiAdmin", "Adminroot", "AdminRoot", "adminRoot", "adminroot", "Rootadmin", "RootAdmin", "rootAdmin", "rootadmin", "Rootroot", "RootRoot", "rootroot", "Administrator", "Password", "Password123", "Password@123", "Password_123", "Password123!", "DDM@123", "ddM@123", "dDm@123",
}

var CHARS = fmt.Sprintf("%s%s%s%s", DIGITS, LETTERS, UPPERS, PUNC)

func RandomPassword2(width int) string {
	for {
		password := randomPassword2(width)
		if !utils.IsInStringArray(password, WEAK_PASSWORDS) {
			return password
		}
	}
}

func randomPassword2(width int) string {
	if width < 6 {
		width = 6
	}
	for {
		ps := PasswordStrength{}
		var buf bytes.Buffer
		for i := 0; i < width; i += 1 {
			var ch byte
			for {
				index := rand.Intn(len(CHARS))
				ch = CHARS[index]
				if i == 0 && ch == '/' {
					continue
				}
				break
			}
			if strings.IndexByte(DIGITS, ch) >= 0 {
				ps.Digits += 1
			} else if strings.IndexByte(LETTERS, ch) >= 0 {
				ps.Lowercases += 1
			} else if strings.IndexByte(UPPERS, ch) >= 0 {
				ps.Uppercases += 1
			} else if strings.IndexByte(PUNC, ch) >= 0 {
				ps.Punctuats += 1
			}
			buf.WriteByte(ch)
		}
		if ps.Digits > 1 && ps.Lowercases > 1 && ps.Uppercases > 1 && ps.Punctuats >= 1 && ps.Punctuats <= 2 {
			return buf.String()
		}
	}
}

func AnalyzePasswordStrenth(passwd string) PasswordStrength {
	ps := PasswordStrength{Invalid: []byte{}}

	for i := 0; i < len(passwd); i += 1 {
		if strings.IndexByte(ALL_DIGITS, passwd[i]) >= 0 {
			ps.Digits += 1
		} else if strings.IndexByte(ALL_LETTERS, passwd[i]) >= 0 {
			ps.Lowercases += 1
		} else if strings.IndexByte(ALL_UPPERS, passwd[i]) >= 0 {
			ps.Uppercases += 1
		} else if strings.IndexByte(ALL_PUNC, passwd[i]) >= 0 {
			ps.Punctuats += 1
		} else {
			ps.Invalid = append(ps.Invalid, passwd[i])
		}
	}
	return ps
}

func (ps PasswordStrength) Len() int {
	return ps.Punctuats + ps.Uppercases + ps.Lowercases + ps.Digits
}

func (ps PasswordStrength) MeetComplexity() bool {
	if ps.Punctuats > 0 && ps.Digits > 0 && ps.Lowercases > 0 && ps.Uppercases > 0 && ps.Len() >= 12 {
		return true
	} else {
		return false
	}
}

func ValidatePassword(passwd string) error {
	ps := AnalyzePasswordStrenth(passwd)
	if len(ps.Invalid) > 0 {
		return httperrors.NewInputParameterError("invalid characters %s", string(ps.Invalid))
	}
	if utils.IsInStringArray(passwd, WEAK_PASSWORDS) || !ps.MeetComplexity() {
		return httperrors.NewWeakPasswordError()
	}
	return nil
}

func MeetComplxity(passwd string) bool {
	if utils.IsInStringArray(passwd, WEAK_PASSWORDS) {
		return false
	}
	ps := AnalyzePasswordStrenth(passwd)
	return ps.MeetComplexity()
}
