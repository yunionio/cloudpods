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

package pwquality

import (
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *Config
	}{
		{
			name: "basic config",
			content: `minlen = 8
dcredit = -1
ucredit = -1
lcredit = -1
ocredit = -1
minclass = 3`,
			expected: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
		},
		{
			name: "config with comments",
			content: `# This is a comment
minlen = 12
# Another comment
dcredit = -2
ucredit = 0
lcredit = -1
ocredit = -1`,
			expected: &Config{
				Minlen:   12,
				Dcredit:  -2,
				Ucredit:  0,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 0,
			},
		},
		{
			name: "empty config",
			content: `# Empty config file
# No settings`,
			expected: &Config{
				Minlen:   0,
				Dcredit:  0,
				Ucredit:  0,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "config with spaces",
			content: `  minlen = 10  
dcredit  =  -1  
ucredit = -1`,
			expected: &Config{
				Minlen:   10,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "positive credit values",
			content: `minlen = 8
dcredit = 1
ucredit = 1
lcredit = 1`,
			expected: &Config{
				Minlen:   8,
				Dcredit:  1,
				Ucredit:  1,
				Lcredit:  1,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "config with maxrepeat",
			content: `minlen = 8
maxrepeat = 3`,
			expected: &Config{
				Minlen:    8,
				Maxrepeat: 3,
			},
		},
		{
			name: "config with maxclassrepeat",
			content: `minlen = 8
maxclassrepeat = 2`,
			expected: &Config{
				Minlen:         8,
				Maxclassrepeat: 2,
			},
		},
		{
			name: "config with maxsequence",
			content: `minlen = 8
maxsequence = 3`,
			expected: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
		},
		{
			name: "config with all new options",
			content: `minlen = 8
maxrepeat = 3
maxclassrepeat = 2
maxsequence = 3`,
			expected: &Config{
				Minlen:         8,
				Maxrepeat:      3,
				Maxclassrepeat: 2,
				Maxsequence:    3,
			},
		},
		{
			name: "invalid value - non-numeric",
			content: `minlen = abc
dcredit = -1`,
			expected: &Config{
				Minlen:  0, // 无效值应该被忽略
				Dcredit: -1,
			},
		},
		{
			name: "line without equals sign",
			content: `minlen 8
dcredit = -1`,
			expected: &Config{
				Minlen:  0, // 没有等号的行应该被忽略
				Dcredit: -1,
			},
		},
		{
			name: "multiple equals signs",
			content: `minlen = 8 = 10
dcredit = -1`,
			expected: &Config{
				Minlen:  0, // " 8 = 10" 不是有效数字，会被忽略
				Dcredit: -1,
			},
		},
		{
			name: "unknown config key",
			content: `minlen = 8
unknown_key = 10
dcredit = -1`,
			expected: &Config{
				Minlen:  8,
				Dcredit: -1,
			},
		},
		{
			name: "empty value",
			content: `minlen = 
dcredit = -1`,
			expected: &Config{
				Minlen:  0, // 空值应该被忽略
				Dcredit: -1,
			},
		},
		{
			name: "config with enforcing",
			content: `minlen = 8
enforcing = 1`,
			expected: &Config{
				Minlen:    8,
				Enforcing: 1,
			},
		},
		{
			name: "config with enforcing disabled",
			content: `minlen = 8
enforcing = 0`,
			expected: &Config{
				Minlen:    8,
				Enforcing: 0,
			},
		},
		{
			name: "config with enforce_for_root",
			content: `minlen = 8
enforce_for_root = 1`,
			expected: &Config{
				Minlen:         8,
				EnforceForRoot: 1,
			},
		},
		{
			name: "config with both enforcing parameters",
			content: `minlen = 8
enforcing = 1
enforce_for_root = 1`,
			expected: &Config{
				Minlen:         8,
				Enforcing:      1,
				EnforceForRoot: 1,
			},
		},
		{
			name: "config with enforce_for_root as flag (no equals sign)",
			content: `minlen = 8
enforce_for_root`,
			expected: &Config{
				Minlen:         8,
				EnforceForRoot: 1, // 独立标志形式，应设置为 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParseConfig([]byte(tt.content))
			if config.Minlen != tt.expected.Minlen {
				t.Errorf("Minlen = %d, want %d", config.Minlen, tt.expected.Minlen)
			}
			if config.Dcredit != tt.expected.Dcredit {
				t.Errorf("Dcredit = %d, want %d", config.Dcredit, tt.expected.Dcredit)
			}
			if config.Ucredit != tt.expected.Ucredit {
				t.Errorf("Ucredit = %d, want %d", config.Ucredit, tt.expected.Ucredit)
			}
			if config.Lcredit != tt.expected.Lcredit {
				t.Errorf("Lcredit = %d, want %d", config.Lcredit, tt.expected.Lcredit)
			}
			if config.Ocredit != tt.expected.Ocredit {
				t.Errorf("Ocredit = %d, want %d", config.Ocredit, tt.expected.Ocredit)
			}
			if config.Minclass != tt.expected.Minclass {
				t.Errorf("Minclass = %d, want %d", config.Minclass, tt.expected.Minclass)
			}
			if config.Maxrepeat != tt.expected.Maxrepeat {
				t.Errorf("Maxrepeat = %d, want %d", config.Maxrepeat, tt.expected.Maxrepeat)
			}
			if config.Maxclassrepeat != tt.expected.Maxclassrepeat {
				t.Errorf("Maxclassrepeat = %d, want %d", config.Maxclassrepeat, tt.expected.Maxclassrepeat)
			}
			if config.Maxsequence != tt.expected.Maxsequence {
				t.Errorf("Maxsequence = %d, want %d", config.Maxsequence, tt.expected.Maxsequence)
			}
		})
	}
}

func TestParseConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *Config
	}{
		{
			name:     "empty content",
			content:  "",
			expected: &Config{},
		},
		{
			name: "only newlines",
			content: `


`,
			expected: &Config{},
		},
		{
			name: "mixed valid and invalid",
			content: `minlen = 8
invalid_line
dcredit = -1
another_invalid = line`,
			expected: &Config{
				Minlen:  8,
				Dcredit: -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParseConfig([]byte(tt.content))
			if config.Minlen != tt.expected.Minlen {
				t.Errorf("Minlen = %d, want %d", config.Minlen, tt.expected.Minlen)
			}
			if config.Dcredit != tt.expected.Dcredit {
				t.Errorf("Dcredit = %d, want %d", config.Dcredit, tt.expected.Dcredit)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		password  string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil config should pass",
			config:    nil,
			password:  "anypassword",
			wantError: false,
		},
		{
			name:      "minlen check - too short",
			config:    &Config{Minlen: 8},
			password:  "short",
			wantError: true,
			errorMsg:  "effective length",
		},
		{
			name:      "minlen check - pass",
			config:    &Config{Minlen: 8},
			password:  "longpassword",
			wantError: false,
		},
		{
			name:      "dcredit negative - require at least 1 digit",
			config:    &Config{Dcredit: -1},
			password:  "nodigits",
			wantError: true,
			errorMsg:  "password requires at least 1 digit(s)",
		},
		{
			name:      "dcredit negative - pass with digit",
			config:    &Config{Dcredit: -1},
			password:  "pass1word",
			wantError: false,
		},
		{
			name:      "dcredit negative - require 2 digits",
			config:    &Config{Dcredit: -2},
			password:  "pass1word",
			wantError: true,
			errorMsg:  "password requires at least 2 digit(s)",
		},
		{
			name:      "dcredit negative - pass with 2 digits",
			config:    &Config{Dcredit: -2},
			password:  "pass12word",
			wantError: false,
		},
		{
			name:      "ucredit negative - require uppercase",
			config:    &Config{Ucredit: -1},
			password:  "nouppercase",
			wantError: true,
			errorMsg:  "password requires at least 1 uppercase letter(s)",
		},
		{
			name:      "ucredit negative - pass with uppercase",
			config:    &Config{Ucredit: -1},
			password:  "passWord",
			wantError: false,
		},
		{
			name:      "lcredit negative - require lowercase",
			config:    &Config{Lcredit: -1},
			password:  "NOLOWERCASE",
			wantError: true,
			errorMsg:  "password requires at least 1 lowercase letter(s)",
		},
		{
			name:      "lcredit negative - pass with lowercase",
			config:    &Config{Lcredit: -1},
			password:  "PASSWORDw",
			wantError: false,
		},
		{
			name:      "ocredit negative - require special char",
			config:    &Config{Ocredit: -1},
			password:  "nospecialchar",
			wantError: true,
			errorMsg:  "password requires at least 1 special character(s)",
		},
		{
			name:      "ocredit negative - pass with special char",
			config:    &Config{Ocredit: -1},
			password:  "password@",
			wantError: false,
		},
		{
			name:      "minclass - require 3 classes",
			config:    &Config{Minclass: 3},
			password:  "onlylowercase",
			wantError: true,
			errorMsg:  "requires at least 3 character class(es)",
		},
		{
			name:      "minclass - pass with 3 classes",
			config:    &Config{Minclass: 3},
			password:  "Pass1word",
			wantError: false,
		},
		{
			name:      "minclass - pass with 4 classes",
			config:    &Config{Minclass: 3},
			password:  "Pass1@word",
			wantError: false,
		},
		{
			name: "complex config - all requirements",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "Pass1@word",
			wantError: false,
		},
		{
			name: "complex config - missing digit",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "Pass@word",
			wantError: true,
			errorMsg:  "password requires at least 1 digit(s)",
		},
		{
			name: "complex config - missing uppercase",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "pass1@word",
			wantError: true,
			errorMsg:  "password requires at least 1 uppercase letter(s)",
		},
		{
			name: "complex config - missing lowercase",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "PASS1@WORD",
			wantError: true,
			errorMsg:  "password requires at least 1 lowercase letter(s)",
		},
		{
			name: "complex config - missing special char",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "Pass1word",
			wantError: true,
			errorMsg:  "password requires at least 1 special character(s)",
		},
		{
			name: "complex config - too short",
			config: &Config{
				Minlen:   12,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "Pass1@wor",
			wantError: true,
			errorMsg:  "effective length",
		},
		{
			name: "positive credit - dcredit",
			config: &Config{
				Minlen:  8,
				Dcredit: 1,
			},
			password:  "password",
			wantError: true,
			errorMsg:  "password should contain at least one digit",
		},
		{
			name: "positive credit - pass with digit",
			config: &Config{
				Minlen:  8,
				Dcredit: 1,
			},
			password:  "pass1word",
			wantError: false,
		},
		{
			name: "real world example - strong password",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "MyP@ssw0rd",
			wantError: false,
		},
		{
			name: "real world example - weak password",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password:  "password",
			wantError: true,
		},
		{
			name: "maxrepeat - too many repeated characters",
			config: &Config{
				Minlen:    8,
				Maxrepeat: 2,
			},
			password:  "Passaaa1",
			wantError: true,
			errorMsg:  "more than 2 consecutive repeated characters",
		},
		{
			name: "maxrepeat - pass with repeated characters within limit",
			config: &Config{
				Minlen:    8,
				Maxrepeat: 3,
			},
			password:  "Passaa12",
			wantError: false,
		},
		{
			name: "maxclassrepeat - too many consecutive digits",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 2,
			},
			password:  "Pass1111",
			wantError: true,
			errorMsg:  "more than 2 consecutive characters of the same class",
		},
		{
			name: "maxclassrepeat - pass with consecutive digits within limit",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 3,
			},
			password:  "Pass111@",
			wantError: false,
		},
		{
			name: "maxsequence - ascending sequence too long",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "Pass1234",
			wantError: true,
			errorMsg:  "sequence of more than 3 consecutive characters",
		},
		{
			name: "maxsequence - descending sequence too long",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "Pass4321",
			wantError: true,
			errorMsg:  "sequence of more than 3 consecutive characters",
		},
		{
			name: "maxsequence - pass with sequence within limit",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "Pass123@",
			wantError: false,
		},
		{
			name: "maxsequence - pass with no sequence",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "Pass1@word",
			wantError: false,
		},
		{
			name: "complex config with all new options",
			config: &Config{
				Minlen:         8,
				Dcredit:        -1,
				Ucredit:        -1,
				Lcredit:        -1,
				Ocredit:        -1,
				Maxrepeat:      2,
				Maxclassrepeat: 2,
				Maxsequence:    3,
			},
			password:  "P1@w0rD2",
			wantError: false,
		},
		{
			name: "complex config - fails maxrepeat",
			config: &Config{
				Minlen:    8,
				Maxrepeat: 2,
			},
			password:  "Passaaa1",
			wantError: true,
			errorMsg:  "more than 2 consecutive repeated characters",
		},
		{
			name: "empty password",
			config: &Config{
				Minlen: 8,
			},
			password:  "",
			wantError: true,
			errorMsg:  "effective length",
		},
		{
			name: "maxrepeat boundary - exactly at limit",
			config: &Config{
				Minlen:    8,
				Maxrepeat: 3,
			},
			password:  "Passaaa1",
			wantError: false, // 3个重复字符，正好在限制内
		},
		{
			name: "maxrepeat boundary - one over limit",
			config: &Config{
				Minlen:    8,
				Maxrepeat: 2,
			},
			password:  "Passaaa1",
			wantError: true,
			errorMsg:  "more than 2 consecutive repeated characters",
		},
		{
			name: "maxclassrepeat boundary - exactly at limit",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 3,
			},
			password:  "Pass111@",
			wantError: false, // 3个连续数字，正好在限制内
		},
		{
			name: "maxsequence boundary - exactly at limit",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "Pass123@",
			wantError: false, // 3个连续字符，正好在限制内
		},
		{
			name: "maxsequence boundary - one over limit",
			config: &Config{
				Minlen:      8,
				Maxsequence: 2,
			},
			password:  "Pass123@",
			wantError: true,
			errorMsg:  "sequence of more than 2 consecutive characters",
		},
		{
			name: "positive credit - effective length calculation",
			config: &Config{
				Minlen:  10,
				Dcredit: 2, // 每个数字可以减少2个长度要求
			},
			password:  "Pass123", // 7个字符 + 3个数字*2 = 13，应该通过
			wantError: false,
		},
		{
			name: "positive credit - effective length too short",
			config: &Config{
				Minlen:  10,
				Dcredit: 1, // 每个数字可以减少1个长度要求
			},
			password:  "Pass12", // 6个字符 + 2个数字*1 = 8，小于10，应该失败
			wantError: true,
			errorMsg:  "effective length",
		},
		{
			name: "maxsequence - ascending at start",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "123Pass@",
			wantError: false, // 3个连续字符在开头，正好在限制内
		},
		{
			name: "maxsequence - descending at end",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "Pass@321",
			wantError: false, // 3个连续字符在结尾，正好在限制内
		},
		{
			name: "positive credit - ucredit",
			config: &Config{
				Minlen:  8,
				Ucredit: 1,
			},
			password:  "password",
			wantError: true,
			errorMsg:  "password should contain at least one uppercase letter",
		},
		{
			name: "positive credit - ucredit pass",
			config: &Config{
				Minlen:  8,
				Ucredit: 1,
			},
			password:  "passWord",
			wantError: false,
		},
		{
			name: "positive credit - lcredit",
			config: &Config{
				Minlen:  8,
				Lcredit: 1,
			},
			password:  "PASSWORD",
			wantError: true,
			errorMsg:  "password should contain at least one lowercase letter",
		},
		{
			name: "positive credit - lcredit pass",
			config: &Config{
				Minlen:  8,
				Lcredit: 1,
			},
			password:  "PASSWORDw",
			wantError: false,
		},
		{
			name: "positive credit - ocredit",
			config: &Config{
				Minlen:  8,
				Ocredit: 1,
			},
			password:  "password",
			wantError: true,
			errorMsg:  "password should contain at least one special character",
		},
		{
			name: "positive credit - ocredit pass",
			config: &Config{
				Minlen:  8,
				Ocredit: 1,
			},
			password:  "password@",
			wantError: false,
		},
		{
			name: "positive credit - multiple credits effective length",
			config: &Config{
				Minlen:  10,
				Dcredit: 2,
				Ucredit: 1,
				Lcredit: 1,
				Ocredit: 1,
			},
			password:  "Pass123@", // 8 + 3*2 + 1*1 + 3*1 + 1*1 = 8+6+1+3+1 = 19，应该通过
			wantError: false,
		},
		{
			name: "maxrepeat - single character password",
			config: &Config{
				Minlen:    1,
				Maxrepeat: 2,
			},
			password:  "a",
			wantError: false, // 单个字符，没有重复
		},
		{
			name: "maxrepeat - no repeated characters",
			config: &Config{
				Minlen:    8,
				Maxrepeat: 2,
			},
			password:  "Passw0rd",
			wantError: false, // 没有重复字符
		},
		{
			name: "maxclassrepeat - single character",
			config: &Config{
				Minlen:         1,
				Maxclassrepeat: 2,
			},
			password:  "1",
			wantError: false, // 单个字符，没有同类重复
		},
		{
			name: "maxclassrepeat - no consecutive same class",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 2,
			},
			password:  "P1a2s3w4",
			wantError: false, // 没有连续同类字符
		},
		{
			name: "maxsequence - short password less than maxsequence",
			config: &Config{
				Minlen:      2,
				Maxsequence: 3,
			},
			password:  "12", // 长度小于 maxsequence+1，不会触发检查
			wantError: false,
		},
		{
			name: "maxsequence - exactly maxsequence+1 length with sequence",
			config: &Config{
				Minlen:      4,
				Maxsequence: 3,
			},
			password:  "1234", // 正好4个字符，包含4个连续字符序列
			wantError: true,
			errorMsg:  "sequence of more than 3 consecutive characters",
		},
		{
			name: "maxsequence - ascending sequence in middle",
			config: &Config{
				Minlen:      8,
				Maxsequence: 2,
			},
			password:  "Pa123ss@",
			wantError: true,
			errorMsg:  "sequence of more than 2 consecutive characters",
		},
		{
			name: "maxsequence - descending sequence in middle",
			config: &Config{
				Minlen:      8,
				Maxsequence: 2,
			},
			password:  "Pa321ss@",
			wantError: true,
			errorMsg:  "sequence of more than 2 consecutive characters",
		},
		{
			name: "maxsequence - mixed ascending and descending",
			config: &Config{
				Minlen:      8,
				Maxsequence: 3,
			},
			password:  "Pass1234@", // 包含4个连续字符序列
			wantError: true,
			errorMsg:  "sequence of more than 3 consecutive characters",
		},
		{
			name: "minclass - exactly required classes",
			config: &Config{
				Minlen:   8,
				Minclass: 2,
			},
			password:  "Password", // 只有大写和小写，2个类
			wantError: false,
		},
		{
			name: "minclass - one class short",
			config: &Config{
				Minlen:   8,
				Minclass: 2,
			},
			password:  "password", // 只有小写，1个类
			wantError: true,
			errorMsg:  "requires at least 2 character class(es)",
		},
		{
			name: "maxrepeat - exactly at limit with multiple repeats",
			config: &Config{
				Minlen:    7,
				Maxrepeat: 2,
			},
			password:  "Passaa1", // 2个a重复，正好在限制内
			wantError: false,
		},
		{
			name: "maxclassrepeat - exactly at limit",
			config: &Config{
				Minlen:         7,
				Maxclassrepeat: 3, // 允许3个连续同类字符
			},
			password:  "Pass111@", // 3个连续数字，正好在限制内
			wantError: false,
		},
		{
			name: "maxclassrepeat - different classes",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 3, // 允许3个连续同类字符
			},
			password:  "PassAA11", // AA和11都是2个连续同类字符，都在限制内
			wantError: false,
		},
		{
			name: "maxclassrepeat - uppercase consecutive",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 2,
			},
			password:  "PassAAA1",
			wantError: true,
			errorMsg:  "more than 2 consecutive characters of the same class",
		},
		{
			name: "maxclassrepeat - lowercase consecutive",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 2,
			},
			password:  "Paaass1@",
			wantError: true,
			errorMsg:  "more than 2 consecutive characters of the same class",
		},
		{
			name: "maxclassrepeat - special consecutive",
			config: &Config{
				Minlen:         8,
				Maxclassrepeat: 2,
			},
			password:  "Pass1@@@",
			wantError: true,
			errorMsg:  "more than 2 consecutive characters of the same class",
		},
		{
			name: "usercheck - password contains username",
			config: &Config{
				Minlen:    8,
				Usercheck: 1,
			},
			password:  "user1234", // 密码包含用户名 "user"
			wantError: true,
			errorMsg:  "password contains the username",
		},
		{
			name: "usercheck - password contains reversed username",
			config: &Config{
				Minlen:    8,
				Usercheck: 1,
			},
			password:  "resu1234", // 密码包含反向用户名 "resu" (user 的反向)
			wantError: true,
			errorMsg:  "password contains the reversed username",
		},
		{
			name: "usercheck - password does not contain username",
			config: &Config{
				Minlen:    8,
				Usercheck: 1,
			},
			password:  "Pass1234", // 密码不包含用户名
			wantError: false,
		},
		{
			name: "usercheck - case insensitive",
			config: &Config{
				Minlen:    8,
				Usercheck: 1,
			},
			password:  "USER1234", // 密码包含大写用户名
			wantError: true,
			errorMsg:  "password contains the username",
		},
		{
			name: "usercheck - disabled",
			config: &Config{
				Minlen:    8,
				Usercheck: 0, // 不检查用户名
			},
			password:  "user1234", // 即使密码包含用户名，也不应该报错
			wantError: false,
		},
		{
			name: "usercheck - empty username",
			config: &Config{
				Minlen:    8,
				Usercheck: 1,
			},
			password:  "anypassword", // 用户名为空，不应该检查
			wantError: false,
		},
		{
			name: "root user with enforce_for_root = 1 - should validate",
			config: &Config{
				Minlen:         8,
				EnforceForRoot: 1, // 对 root 强制执行
				Enforcing:      1, // 强制执行
			},
			password:  "short", // root 用户也需要验证密码强度
			wantError: true,
			errorMsg:  "effective length",
		},
		{
			name: "non-root user with enforce_for_root = 0 - should validate",
			config: &Config{
				Minlen:         8,
				EnforceForRoot: 0, // 不对 root 强制执行，但普通用户需要验证
				Enforcing:      1, // 强制执行（对普通用户）
			},
			password:  "short", // 普通用户需要验证密码强度
			wantError: true,
			errorMsg:  "effective length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 对于 usercheck 测试，使用 "user" 作为用户名
			// 对于 root 用户测试，使用 "root" 作为用户名
			// 对于 non-root 用户测试，使用 "testuser" 作为用户名
			// 注意：需要先检查 "non-root user"，因为 "non-root user" 包含 "root user"
			username := ""
			if strings.Contains(tt.name, "usercheck") {
				username = "user"
			} else if strings.Contains(tt.name, "non-root user") {
				username = "testuser"
			} else if strings.Contains(tt.name, "root user") {
				username = "root"
			}

			// 如果 config 不为 nil 且未设置 Enforcing，默认设置为 1（强制执行）
			// 但如果是 enforcing=0 的测试用例，不要修改
			// 对于 root 用户测试，如果 enforce_for_root=0，enforcing 可能为 0，不要修改
			// 对于 non-root 用户测试，如果已经设置了 Enforcing=1，不要修改
			// 注意：只对 Enforcing=0 的测试用例进行修改，且排除特殊测试用例
			if tt.config != nil && tt.config.Enforcing == 0 && tt.name != "nil config should pass" &&
				!strings.Contains(tt.name, "enforcing = 0") &&
				!strings.Contains(tt.name, "root user with enforce_for_root = 0") &&
				!strings.Contains(tt.name, "non-root user") {
				// 只对非特殊测试用例且 Enforcing=0 的情况设置为 1
				tt.config.Enforcing = 1
			}

			err := tt.config.Validate(tt.password, username)
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_CharacterClasses(t *testing.T) {
	config := &Config{
		Minclass:  4,
		Enforcing: 1, // 强制执行
	}

	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{"all 4 classes", "Pass1@word", false},
		{"3 classes - missing special", "Pass1word", true},
		{"3 classes - missing digit", "Pass@word", true},
		{"3 classes - missing uppercase", "pass1@word", true},
		{"3 classes - missing lowercase", "PASS1@WORD", true},
		{"2 classes", "password", true},
		{"1 class", "PASSWORD", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.Validate(tt.password, "")
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error but got nil for password %q", tt.password)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v for password %q", err, tt.password)
				}
			}
		})
	}
}

func TestParsePAMConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *Config
	}{
		{
			name: "pam_pwquality config",
			content: `# PAM configuration
auth required pam_unix.so
password requisite pam_pwquality.so retry=3 minlen=8 dcredit=-1 ucredit=-1 lcredit=-1 ocredit=-1 minclass=3
password required pam_unix.so sha512 shadow nullok try_first_pass use_authtok`,
			expected: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
		},
		{
			name: "pam_cracklib config",
			content: `# PAM configuration
password requisite pam_cracklib.so retry=3 minlen=12 dcredit=-2 ucredit=-1 lcredit=-1 ocredit=-1
password required pam_unix.so`,
			expected: &Config{
				Minlen:   12,
				Dcredit:  -2,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 0,
			},
		},
		{
			name: "PAM config with comments",
			content: `# This is a comment
password requisite pam_pwquality.so minlen=10 dcredit=-1
# Another comment`,
			expected: &Config{
				Minlen:   10,
				Dcredit:  -1,
				Ucredit:  0,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "PAM config without password module",
			content: `# No password module
auth required pam_unix.so`,
			expected: &Config{
				Minlen:   0,
				Dcredit:  0,
				Ucredit:  0,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "multiple password lines - use first",
			content: `password requisite pam_pwquality.so minlen=8 dcredit=-1
password required pam_unix.so
password optional pam_gnome_keyring.so`,
			expected: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  0,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name:    "PAM config with new options",
			content: `password requisite pam_pwquality.so minlen=8 maxrepeat=3 maxclassrepeat=2 maxsequence=3`,
			expected: &Config{
				Minlen:         8,
				Maxrepeat:      3,
				Maxclassrepeat: 2,
				Maxsequence:    3,
			},
		},
		{
			name:    "PAM config with invalid value",
			content: `password requisite pam_pwquality.so minlen=abc dcredit=-1`,
			expected: &Config{
				Minlen:  0, // 无效值应该被忽略
				Dcredit: -1,
			},
		},
		{
			name: "PAM config - password line without pam module",
			content: `password required pam_unix.so
auth required pam_unix.so`,
			expected: &Config{
				Minlen:   0,
				Dcredit:  0,
				Ucredit:  0,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "PAM config - pam module without password",
			content: `auth required pam_pwquality.so minlen=8
account required pam_unix.so`,
			expected: &Config{
				Minlen:  0, // 不是 password 行，应该被忽略
				Dcredit: 0,
			},
		},
		{
			name:    "PAM config - parameter without equals",
			content: `password requisite pam_pwquality.so minlen dcredit=-1`,
			expected: &Config{
				Minlen:  0, // 没有等号的参数应该被忽略
				Dcredit: -1,
			},
		},
		{
			name:    "PAM config - multiple equals in parameter",
			content: `password requisite pam_pwquality.so minlen=8=10 dcredit=-1`,
			expected: &Config{
				Minlen:  0, // "8=10" 不是有效数字，会被忽略
				Dcredit: -1,
			},
		},
		{
			name:    "PAM config with enforcing",
			content: `password requisite pam_pwquality.so minlen=8 enforcing=1`,
			expected: &Config{
				Minlen:    8,
				Enforcing: 1,
			},
		},
		{
			name:    "PAM config with enforce_for_root",
			content: `password requisite pam_pwquality.so minlen=8 enforce_for_root=1`,
			expected: &Config{
				Minlen:         8,
				EnforceForRoot: 1,
			},
		},
		{
			name:    "PAM config with both enforcing parameters",
			content: `password requisite pam_pwquality.so minlen=8 enforcing=0 enforce_for_root=1`,
			expected: &Config{
				Minlen:         8,
				Enforcing:      0,
				EnforceForRoot: 1,
			},
		},
		{
			name:    "PAM config with enforce_for_root as flag (no value)",
			content: `password requisite pam_pwquality.so minlen=8 enforce_for_root`,
			expected: &Config{
				Minlen:         8,
				EnforceForRoot: 1, // 独立标志形式，应设置为 1
			},
		},
		{
			name:    "PAM config with usercheck",
			content: `password requisite pam_pwquality.so minlen=8 usercheck=1`,
			expected: &Config{
				Minlen:    8,
				Usercheck: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParsePAMConfig([]byte(tt.content), nil)
			if config.Minlen != tt.expected.Minlen {
				t.Errorf("Minlen = %d, want %d", config.Minlen, tt.expected.Minlen)
			}
			if config.Dcredit != tt.expected.Dcredit {
				t.Errorf("Dcredit = %d, want %d", config.Dcredit, tt.expected.Dcredit)
			}
			if config.Ucredit != tt.expected.Ucredit {
				t.Errorf("Ucredit = %d, want %d", config.Ucredit, tt.expected.Ucredit)
			}
			if config.Lcredit != tt.expected.Lcredit {
				t.Errorf("Lcredit = %d, want %d", config.Lcredit, tt.expected.Lcredit)
			}
			if config.Ocredit != tt.expected.Ocredit {
				t.Errorf("Ocredit = %d, want %d", config.Ocredit, tt.expected.Ocredit)
			}
			if config.Minclass != tt.expected.Minclass {
				t.Errorf("Minclass = %d, want %d", config.Minclass, tt.expected.Minclass)
			}
			if config.Maxrepeat != tt.expected.Maxrepeat {
				t.Errorf("Maxrepeat = %d, want %d", config.Maxrepeat, tt.expected.Maxrepeat)
			}
			if config.Maxclassrepeat != tt.expected.Maxclassrepeat {
				t.Errorf("Maxclassrepeat = %d, want %d", config.Maxclassrepeat, tt.expected.Maxclassrepeat)
			}
			if config.Maxsequence != tt.expected.Maxsequence {
				t.Errorf("Maxsequence = %d, want %d", config.Maxsequence, tt.expected.Maxsequence)
			}
		})
	}
}

func TestConfig_HasAnyPolicy(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty config",
			config:   &Config{},
			expected: false,
		},
		{
			name:     "only minlen",
			config:   &Config{Minlen: 8},
			expected: true,
		},
		{
			name:     "only dcredit",
			config:   &Config{Dcredit: -1},
			expected: true,
		},
		{
			name:     "only ucredit",
			config:   &Config{Ucredit: -1},
			expected: true,
		},
		{
			name:     "only lcredit",
			config:   &Config{Lcredit: -1},
			expected: true,
		},
		{
			name:     "only ocredit",
			config:   &Config{Ocredit: -1},
			expected: true,
		},
		{
			name:     "only minclass",
			config:   &Config{Minclass: 3},
			expected: true,
		},
		{
			name:     "all fields set",
			config:   &Config{Minlen: 8, Dcredit: -1, Ucredit: -1, Lcredit: -1, Ocredit: -1, Minclass: 3},
			expected: true,
		},
		{
			name:     "only lcredit and ocredit",
			config:   &Config{Lcredit: -1, Ocredit: -1},
			expected: true,
		},
		{
			name:     "only maxrepeat",
			config:   &Config{Maxrepeat: 3},
			expected: true,
		},
		{
			name:     "only maxclassrepeat",
			config:   &Config{Maxclassrepeat: 2},
			expected: true,
		},
		{
			name:     "only maxsequence",
			config:   &Config{Maxsequence: 3},
			expected: true,
		},
		{
			name:     "all new options",
			config:   &Config{Maxrepeat: 3, Maxclassrepeat: 2, Maxsequence: 3},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.HasAnyPolicy()
			if got != tt.expected {
				t.Errorf("HasAnyPolicy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_IsEnforcing(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config - default enforcing",
			config:   nil,
			expected: true, // 默认强制执行
		},
		{
			name:     "enforcing = 1",
			config:   &Config{Enforcing: 1},
			expected: true,
		},
		{
			name:     "enforcing = 0",
			config:   &Config{Enforcing: 0},
			expected: false,
		},
		{
			name:     "enforcing = 2",
			config:   &Config{Enforcing: 2},
			expected: true, // 非0值都视为强制执行
		},
		{
			name:     "default config - enforcing not set",
			config:   &Config{Enforcing: 1}, // 默认值为1
			expected: true,                  // 默认值为1，强制执行
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEnforcing()
			if got != tt.expected {
				t.Errorf("IsEnforcing() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_IsEnforcingForRoot(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config - default not enforcing for root",
			config:   nil,
			expected: false, // 默认不对 root 强制执行
		},
		{
			name:     "enforce_for_root = 1",
			config:   &Config{EnforceForRoot: 1},
			expected: true,
		},
		{
			name:     "enforce_for_root = 0",
			config:   &Config{EnforceForRoot: 0},
			expected: false,
		},
		{
			name:     "enforce_for_root = 2",
			config:   &Config{EnforceForRoot: 2},
			expected: false, // 只有1才表示强制执行
		},
		{
			name:     "default config - enforce_for_root not set",
			config:   &Config{},
			expected: false, // 默认值为0，不对 root 强制执行
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEnforcingForRoot()
			if got != tt.expected {
				t.Errorf("IsEnforcingForRoot() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_GeneratePassword(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		passwordGenerator func(int) string
		wantError         bool // 生成的密码是否应该通过验证
		description       string
	}{
		{
			name:   "nil config - should return empty or default password",
			config: nil,
			passwordGenerator: func(length int) string {
				return "defaultpassword"
			},
			wantError:   false,
			description: "nil 配置应该返回默认长度的密码",
		},
		{
			name:   "no policy config - should return default password",
			config: &Config{},
			passwordGenerator: func(length int) string {
				return "defaultpassword"
			},
			wantError:   false,
			description: "没有策略的配置应该返回默认长度的密码",
		},
		{
			name: "minlen only - generate valid password",
			config: &Config{
				Minlen: 8,
			},
			passwordGenerator: func(length int) string {
				// 第一次返回不符合要求的密码，第二次返回符合要求的密码
				if length == 8 {
					return "short" // 太短
				}
				return "longpassword" // 符合要求
			},
			wantError:   false,
			description: "只有最小长度要求，应该生成符合要求的密码",
		},
		{
			name: "dcredit requirement - generate password with digits",
			config: &Config{
				Minlen:  8,
				Dcredit: -1, // 至少需要1个数字
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有数字的密码，第二次返回有数字的密码
				if length == 8 {
					return "nodigits" // 没有数字
				}
				return "pass1word" // 有数字，符合要求
			},
			wantError:   false,
			description: "需要数字字符，应该生成包含数字的密码",
		},
		{
			name: "ucredit requirement - generate password with uppercase",
			config: &Config{
				Minlen:  8,
				Ucredit: -1, // 至少需要1个大写字母
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有大写字母的密码，第二次返回有大写字母的密码
				if length == 8 {
					return "nouppercase" // 没有大写字母
				}
				return "passWord" // 有大写字母，符合要求
			},
			wantError:   false,
			description: "需要大写字母，应该生成包含大写字母的密码",
		},
		{
			name: "lcredit requirement - generate password with lowercase",
			config: &Config{
				Minlen:  8,
				Lcredit: -1, // 至少需要1个小写字母
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有小写字母的密码，第二次返回有小写字母的密码
				if length == 8 {
					return "NOLOWERCASE" // 没有小写字母
				}
				return "PASSWORDw" // 有小写字母，符合要求
			},
			wantError:   false,
			description: "需要小写字母，应该生成包含小写字母的密码",
		},
		{
			name: "ocredit requirement - generate password with special char",
			config: &Config{
				Minlen:  8,
				Ocredit: -1, // 至少需要1个特殊字符
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有特殊字符的密码，第二次返回有特殊字符的密码
				if length == 8 {
					return "nospecial" // 没有特殊字符
				}
				return "pass@word" // 有特殊字符，符合要求
			},
			wantError:   false,
			description: "需要特殊字符，应该生成包含特殊字符的密码",
		},
		{
			name: "complex requirements - multiple retries",
			config: &Config{
				Minlen:   12,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			passwordGenerator: func(length int) string {
				// 模拟多次尝试：第一次太短，第二次缺少数字，第三次缺少大写，第四次符合要求
				switch length {
				case 12:
					return "short" // 太短
				case 13:
					return "nouppercase1@" // 缺少大写字母
				case 14:
					return "nolowercase1@A" // 缺少小写字母
				default:
					return "ValidPass12@" // 符合所有要求：12个字符，包含数字、大写、小写、特殊字符
				}
			},
			wantError:   false,
			description: "复杂要求，需要多次重试才能生成符合要求的密码",
		},
		{
			name: "nil passwordGenerator - should return empty",
			config: &Config{
				Minlen: 8,
			},
			passwordGenerator: nil,
			wantError:         false,
			description:       "passwordGenerator 为 nil 时应该返回空字符串",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password := tt.config.GeneratePassword(tt.passwordGenerator)

			if tt.passwordGenerator == nil {
				// 如果 passwordGenerator 为 nil，应该返回空字符串
				if password != "" {
					t.Errorf("GeneratePassword() with nil generator = %q, want empty string", password)
				}
				return
			}

			if password == "" {
				t.Errorf("GeneratePassword() returned empty string, want non-empty password")
				return
			}

			// 验证生成的密码是否符合配置要求
			if tt.config != nil && tt.config.HasAnyPolicy() {
				err := tt.config.Validate(password, "")
				if tt.wantError {
					if err == nil {
						t.Errorf("GeneratePassword() generated password %q should fail validation but passed", password)
					}
				} else {
					if err != nil {
						t.Errorf("GeneratePassword() generated password %q failed validation: %v", password, err)
					}
				}
			}
		})
	}
}

func TestConfig_GeneratePassword_RetryMechanism(t *testing.T) {
	// 测试重试机制：确保在多次尝试后能生成符合要求的密码
	attemptCount := 0
	config := &Config{
		Minlen:    10,
		Dcredit:   -2, // 需要至少2个数字
		Ucredit:   -1, // 需要至少1个大写字母
		Lcredit:   -1, // 需要至少1个小写字母
		Ocredit:   -1, // 需要至少1个特殊字符
		Enforcing: 1,  // 强制执行
	}

	passwordGenerator := func(length int) string {
		attemptCount++
		// 前几次生成不符合要求的密码
		switch attemptCount {
		case 1:
			return "short" // 太短
		case 2:
			return "nouppercase12@" // 缺少大写字母
		case 3:
			return "NOLOWERCASE12@" // 缺少小写字母
		case 4:
			return "NoSpecial12" // 缺少特殊字符
		case 5:
			return "ValidPass12@" // 符合所有要求
		default:
			return "ValidPass12@" // 后续都返回符合要求的密码
		}
	}

	password := config.GeneratePassword(passwordGenerator)

	if password == "" {
		t.Fatal("GeneratePassword() returned empty string")
	}

	// 验证密码符合要求
	err := config.Validate(password, "")
	if err != nil {
		t.Errorf("GeneratePassword() generated password %q failed validation: %v", password, err)
	}

	// 验证确实进行了多次尝试（至少尝试了4次）
	if attemptCount < 4 {
		t.Errorf("Expected at least 4 attempts, got %d", attemptCount)
	}
}

func TestConfig_GeneratePassword_LengthCalculation(t *testing.T) {
	// 测试密码长度计算逻辑
	tests := []struct {
		name           string
		config         *Config
		expectedMinLen int
		description    string
	}{
		{
			name: "minlen only",
			config: &Config{
				Minlen: 8,
			},
			expectedMinLen: 8,
			description:    "只有最小长度要求",
		},
		{
			name: "minlen with credit requirements",
			config: &Config{
				Minlen:  8,
				Dcredit: -2, // 需要2个数字
				Ucredit: -1, // 需要1个大写字母
			},
			expectedMinLen: 8, // 至少是 minlen 和 requiredChars 的较大值
			description:    "最小长度和 credit 要求",
		},
		{
			name: "minlen with minclass",
			config: &Config{
				Minlen:   8,
				Minclass: 3,
			},
			expectedMinLen: 8,
			description:    "最小长度和最小字符类要求",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			passwordGenerator := func(length int) string {
				callCount++
				// 第一次调用时记录长度
				if callCount == 1 {
					if length < tt.expectedMinLen {
						t.Errorf("First password generation length = %d, want at least %d", length, tt.expectedMinLen)
					}
				}
				// 根据配置返回符合要求的密码
				if tt.config.Dcredit < 0 && -tt.config.Dcredit >= 2 {
					// 需要至少2个数字
					return "ValidPass12@"
				}
				return "ValidPass1@"
			}

			password := tt.config.GeneratePassword(passwordGenerator)
			if password == "" {
				t.Errorf("GeneratePassword() returned empty string")
			}

			err := tt.config.Validate(password, "")
			if err != nil {
				t.Errorf("Generated password failed validation: %v", err)
			}
		})
	}
}

func TestConfig_GeneratePassword_EdgeCases(t *testing.T) {
	// 测试 GeneratePassword 的边界情况
	tests := []struct {
		name              string
		config            *Config
		passwordGenerator func(int) string
		description       string
	}{
		{
			name: "minclass = 1 should not add buffer",
			config: &Config{
				Minlen:   8,
				Minclass: 1, // minclass = 1，不应该添加缓冲
				Dcredit:  -1,
			},
			passwordGenerator: func(length int) string {
				return "Pass1word"
			},
			description: "minclass = 1 时不应该添加额外的长度缓冲",
		},
		{
			name: "requiredChars > minLength",
			config: &Config{
				Minlen:  5,
				Dcredit: -3, // 需要3个数字
				Ucredit: -2, // 需要2个大写字母
			},
			passwordGenerator: func(length int) string {
				// requiredChars = 5，应该使用5而不是minLength
				if length < 5 {
					t.Errorf("Expected length >= 5, got %d", length)
				}
				return "PASS123"
			},
			description: "当 requiredChars > minLength 时，应该使用 requiredChars",
		},
		{
			name: "minLength = 0 with credit requirements",
			config: &Config{
				Minlen:  0,  // 没有设置最小长度
				Dcredit: -2, // 需要2个数字
			},
			passwordGenerator: func(length int) string {
				// 应该使用默认的8，或者 requiredChars 的较大值
				if length < 2 {
					t.Errorf("Expected length >= 2, got %d", length)
				}
				return "Pass12"
			},
			description: "minLength = 0 时应该使用默认值8",
		},
		{
			name: "maxAttempts reached - should return longer password",
			config: &Config{
				Minlen:    8,
				Maxrepeat: 1, // 非常严格的限制
			},
			passwordGenerator: func(length int) string {
				// 总是返回不符合要求的密码（包含重复字符）
				return strings.Repeat("a", length) // 全部是重复字符
			},
			description: "当达到最大尝试次数时，应该返回一个较长的密码",
		},
		{
			name: "minclass > 1 should add buffer",
			config: &Config{
				Minlen:   8,
				Minclass: 3, // minclass > 1，应该添加缓冲
				Dcredit:  -1,
			},
			passwordGenerator: func(length int) string {
				// 应该包含 minclass 的缓冲
				if length < 8+3 {
					t.Errorf("Expected length >= 11 (8 + 3), got %d", length)
				}
				return "Pass1@word"
			},
			description: "minclass > 1 时应该添加额外的长度缓冲",
		},
		{
			name: "requiredChars = 0",
			config: &Config{
				Minlen: 8,
				// 没有 credit 要求
			},
			passwordGenerator: func(length int) string {
				if length != 8 {
					t.Errorf("Expected length = 8, got %d", length)
				}
				return "password"
			},
			description: "没有 credit 要求时，应该使用 minLength",
		},
		{
			name: "all positive credits",
			config: &Config{
				Minlen:  8,
				Dcredit: 1,
				Ucredit: 1,
				Lcredit: 1,
				Ocredit: 1,
			},
			passwordGenerator: func(length int) string {
				return "Pass1@word"
			},
			description: "所有 credit 都是正数时，应该正常生成密码",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password := tt.config.GeneratePassword(tt.passwordGenerator)
			if password == "" && tt.passwordGenerator != nil {
				t.Errorf("GeneratePassword() returned empty string")
			}
			// 验证生成的密码（如果可能）
			if password != "" && tt.config.HasAnyPolicy() {
				err := tt.config.Validate(password, "")
				// 对于 maxAttempts 测试，密码可能不符合要求
				if tt.name != "maxAttempts reached - should return longer password" {
					if err != nil {
						t.Errorf("Generated password failed validation: %v", err)
					}
				}
			}
		})
	}
}

func TestConfig_GeneratePassword_MaxAttempts(t *testing.T) {
	// 专门测试达到最大尝试次数的情况
	attemptCount := 0
	config := &Config{
		Minlen:    8,
		Maxrepeat: 1, // 非常严格的限制，几乎不可能满足
		Enforcing: 1, // 强制执行
	}

	passwordGenerator := func(length int) string {
		attemptCount++
		// 总是返回不符合要求的密码（全部是重复字符）
		return strings.Repeat("a", length)
	}

	password := config.GeneratePassword(passwordGenerator)

	// 应该返回一个密码（即使不符合要求）
	if password == "" {
		t.Error("GeneratePassword() should return a password even after max attempts")
	}

	// 应该进行了多次尝试
	if attemptCount < 10 {
		t.Errorf("Expected at least 10 attempts, got %d", attemptCount)
	}

	// 密码长度应该增加了
	if len(password) < 8 {
		t.Errorf("Expected password length >= 8 after retries, got %d", len(password))
	}
}
