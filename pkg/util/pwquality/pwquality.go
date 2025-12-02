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
	"strconv"
	"strings"
	"unicode"

	"yunion.io/x/pkg/errors"
)

// ErrPasswordTooWeak 表示密码强度不符合要求的统一错误
var ErrPasswordTooWeak = errors.Error("password too weak")

// Config 存储 pwquality 配置
type Config struct {
	Minlen         int // 最小长度
	Dcredit        int // 数字字符信用值（负数表示至少需要多少个字符，正数表示每个字符可减少的长度要求）
	Ucredit        int // 大写字母信用值
	Lcredit        int // 小写字母信用值
	Ocredit        int // 特殊字符信用值
	Minclass       int // 最小字符类数量（数字、大写、小写、特殊）
	Maxrepeat      int // 最大重复字符数（0 表示不限制）
	Maxclassrepeat int // 最大同类字符重复数（0 表示不限制）
	Maxsequence    int // 最大连续字符序列长度（0 表示不限制）
	// Enforcing 是否强制执行密码策略（1=强制执行，0=仅警告，默认1）
	// 注意：此参数在 libpwquality 1.2.0+ 版本中支持，较老的系统可能不支持
	// 如果系统不支持，配置文件中不会出现此参数，将使用默认值 1（强制执行）
	Enforcing int
	// EnforceForRoot 是否对 root 用户强制执行密码策略（1=强制执行，0=不强制，默认0）
	// 注意：此参数在 libpwquality 1.2.0+ 版本中支持，较老的系统可能不支持
	// 在配置文件中，可能以两种形式出现：
	//   1. enforce_for_root = 1（key=value 形式）
	//   2. enforce_for_root（独立标志形式，无等号，表示启用）
	// 如果系统不支持，配置文件中不会出现此参数，将使用默认值 0（不对 root 强制执行）
	EnforceForRoot int
	Usercheck      int // 是否检查密码中包含用户名（1=检查，0=不检查，默认0）
	// 以下配置项在 chroot 环境中可能不适用，暂不实现
	// Gecoscheck int // 是否检查密码中包含用户的 GECOS 信息
	// Dictcheck  int // 是否检查密码是否包含字典中的单词（需要字典文件）
	// Dictpath   string // 字典文件路径
}

// HasAnyPolicy 检查配置是否有任何非默认的密码策略设置
// 用于判断配置是否有效（即是否包含任何密码强度要求）
func (c *Config) HasAnyPolicy() bool {
	if c == nil {
		return false
	}
	return c.Minlen > 0 || c.Dcredit != 0 || c.Ucredit != 0 ||
		c.Lcredit != 0 || c.Ocredit != 0 || c.Minclass > 0 ||
		c.Maxrepeat > 0 || c.Maxclassrepeat > 0 || c.Maxsequence > 0
}

// IsEnforcing 检查密码策略是否强制执行
// 如果 enforcing=0，密码策略不会强制执行（只是警告）
func (c *Config) IsEnforcing() bool {
	if c == nil {
		return true // 默认强制执行
	}
	// enforcing=1 表示强制执行，enforcing=0 表示仅警告
	// 默认值为 1（强制执行）
	return c.Enforcing != 0
}

// IsEnforcingForRoot 检查是否对 root 用户强制执行密码策略
func (c *Config) IsEnforcingForRoot() bool {
	if c == nil {
		return false // 默认不对 root 强制执行
	}
	// enforce_for_root=1 表示对 root 强制执行，enforce_for_root=0 表示不强制
	return c.EnforceForRoot == 1
}

// ParseConfig 解析 /etc/security/pwquality.conf 配置文件内容
//
// 兼容性说明：
// - enforcing 和 enforce_for_root 参数在 libpwquality 1.2.0+ 版本中支持
// - 较老的系统（如 RHEL 6 之前）可能不支持这些参数
// - enforce_for_root 可能以两种形式出现：
//  1. enforce_for_root = 1（key=value 形式）
//  2. enforce_for_root（独立标志形式，无等号，表示启用）
//
// - 如果配置文件中不存在这些参数，将使用默认值：
//   - Enforcing: 1（默认强制执行）
//   - EnforceForRoot: 0（默认不对 root 强制执行）
func ParseConfig(content []byte) *Config {
	config := &Config{
		Minlen:         0, // 默认值
		Dcredit:        0, // 默认值
		Ucredit:        0, // 默认值
		Lcredit:        0, // 默认值
		Ocredit:        0, // 默认值
		Minclass:       0, // 默认值
		Maxrepeat:      0, // 默认值（0 表示不限制）
		Maxclassrepeat: 0, // 默认值（0 表示不限制）
		Maxsequence:    0, // 默认值（0 表示不限制）
		Enforcing:      1, // 默认值（1 表示强制执行）
		EnforceForRoot: 0, // 默认值（0 表示不对 root 强制执行）
		Usercheck:      0, // 默认值（0 表示不检查用户名）
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过注释和空行
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是 key = value 格式
		if strings.Contains(line, "=") {
			// 解析 key = value 格式
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "minlen":
				if v, err := strconv.Atoi(value); err == nil {
					config.Minlen = v
				}
			case "dcredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Dcredit = v
				}
			case "ucredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Ucredit = v
				}
			case "lcredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Lcredit = v
				}
			case "ocredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Ocredit = v
				}
			case "minclass":
				if v, err := strconv.Atoi(value); err == nil {
					config.Minclass = v
				}
			case "maxrepeat":
				if v, err := strconv.Atoi(value); err == nil {
					config.Maxrepeat = v
				}
			case "maxclassrepeat":
				if v, err := strconv.Atoi(value); err == nil {
					config.Maxclassrepeat = v
				}
			case "maxsequence":
				if v, err := strconv.Atoi(value); err == nil {
					config.Maxsequence = v
				}
			case "enforcing":
				if v, err := strconv.Atoi(value); err == nil {
					config.Enforcing = v
				}
			case "enforce_for_root":
				if v, err := strconv.Atoi(value); err == nil {
					config.EnforceForRoot = v
				}
			case "usercheck":
				if v, err := strconv.Atoi(value); err == nil {
					config.Usercheck = v
				}
			}
		} else {
			// 处理独立标志（无值的参数）
			// 例如：enforce_for_root（表示对 root 强制执行密码策略）
			key := strings.TrimSpace(line)
			switch key {
			case "enforce_for_root":
				// 如果以独立标志形式出现，设置为 1（强制执行）
				config.EnforceForRoot = 1
			}
		}
	}

	return config
}

// Validate 根据 pwquality 配置校验密码强度
// username 为用户名，用于检查密码中是否包含用户名（如果启用了 usercheck）
// 参考 libpwquality 的实现逻辑
func (c *Config) Validate(password string, username string) error {
	if !c.HasAnyPolicy() {
		return nil
	}

	// 如果 enforcing=0，密码策略不会强制执行（只是警告），直接返回
	//if username != "root" && !c.IsEnforcing() {
	//	return nil
	//}

	// 如果用户是 root 且 enforce_for_root=0，不对 root 强制执行密码策略
	//if username == "root" && !c.IsEnforcingForRoot() {
	//	return nil
	//}

	// 统计各类字符数量
	var digits, uppers, lowers, others int
	for _, r := range password {
		if unicode.IsDigit(r) {
			digits++
		} else if unicode.IsUpper(r) {
			uppers++
		} else if unicode.IsLower(r) {
			lowers++
		} else {
			others++
		}
	}

	// 处理 credit 值
	// 根据 libpwquality 的文档：
	// - 负数（如 -1）：表示至少需要多少个字符（常用）
	// - 正数（如 1）：表示每个字符可以减少多少长度要求（较少用）
	// - 0：不要求
	if c.Dcredit < 0 {
		// 负数：至少需要这么多数字字符
		required := -c.Dcredit
		if digits < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d digit(s), got %d", required, digits)
		}
	} else if c.Dcredit > 0 {
		// 正数：每个数字字符可以减少多少长度要求（较少使用）
		// 这里我们简化处理，只检查是否有数字
		if digits == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one digit")
		}
	}

	if c.Ucredit < 0 {
		required := -c.Ucredit
		if uppers < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d uppercase letter(s), got %d", required, uppers)
		}
	} else if c.Ucredit > 0 {
		if uppers == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one uppercase letter")
		}
	}

	if c.Lcredit < 0 {
		required := -c.Lcredit
		if lowers < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d lowercase letter(s), got %d", required, lowers)
		}
	} else if c.Lcredit > 0 {
		if lowers == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one lowercase letter")
		}
	}

	if c.Ocredit < 0 {
		required := -c.Ocredit
		if others < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d special character(s), got %d", required, others)
		}
	} else if c.Ocredit > 0 {
		if others == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one special character")
		}
	}

	// 计算有效长度（考虑 credit 的正数值，用于减少长度要求）
	effectiveLength := len(password)
	if c.Dcredit > 0 {
		effectiveLength += digits * c.Dcredit
	}
	if c.Ucredit > 0 {
		effectiveLength += uppers * c.Ucredit
	}
	if c.Lcredit > 0 {
		effectiveLength += lowers * c.Lcredit
	}
	if c.Ocredit > 0 {
		effectiveLength += others * c.Ocredit
	}

	// 检查最小长度
	if c.Minlen > 0 && effectiveLength < c.Minlen {
		return errors.Wrapf(ErrPasswordTooWeak, "effective length %d is less than required %d", effectiveLength, c.Minlen)
	}

	// 检查最小字符类数量
	if c.Minclass > 0 {
		classes := 0
		if digits > 0 {
			classes++
		}
		if uppers > 0 {
			classes++
		}
		if lowers > 0 {
			classes++
		}
		if others > 0 {
			classes++
		}
		if classes < c.Minclass {
			return errors.Wrapf(ErrPasswordTooWeak, "requires at least %d character class(es), got %d", c.Minclass, classes)
		}
	}

	// 检查最大重复字符数
	if c.Maxrepeat > 0 {
		maxRepeat := 0
		currentRepeat := 1
		prevChar := rune(0)
		for _, r := range password {
			if r == prevChar {
				currentRepeat++
				if currentRepeat > maxRepeat {
					maxRepeat = currentRepeat
				}
			} else {
				currentRepeat = 1
			}
			prevChar = r
		}
		if maxRepeat > c.Maxrepeat {
			return errors.Wrapf(ErrPasswordTooWeak, "password contains more than %d consecutive repeated characters", c.Maxrepeat)
		}
	}

	// 检查最大同类字符重复数
	if c.Maxclassrepeat > 0 {
		maxClassRepeat := 0
		currentClassRepeat := 1
		prevClass := -1 // -1: 未设置, 0: 数字, 1: 大写, 2: 小写, 3: 特殊
		for _, r := range password {
			var currentClass int
			if unicode.IsDigit(r) {
				currentClass = 0
			} else if unicode.IsUpper(r) {
				currentClass = 1
			} else if unicode.IsLower(r) {
				currentClass = 2
			} else {
				currentClass = 3
			}
			if currentClass == prevClass {
				currentClassRepeat++
				if currentClassRepeat > maxClassRepeat {
					maxClassRepeat = currentClassRepeat
				}
			} else {
				currentClassRepeat = 1
			}
			prevClass = currentClass
		}
		if maxClassRepeat > c.Maxclassrepeat {
			return errors.Wrapf(ErrPasswordTooWeak, "password contains more than %d consecutive characters of the same class", c.Maxclassrepeat)
		}
	}

	// 检查最大连续字符序列长度
	// maxsequence 检查密码中是否存在超过指定长度的连续字符序列（如 "1234" 或 "abcd"）
	if c.Maxsequence > 0 {
		runes := []rune(password)
		for i := 0; i <= len(runes)-c.Maxsequence-1; i++ {
			// 检查升序序列（如 "1234", "abcd"）
			isAscending := true
			for j := 1; j <= c.Maxsequence; j++ {
				if i+j >= len(runes) || runes[i+j] != runes[i+j-1]+1 {
					isAscending = false
					break
				}
			}
			// 检查降序序列（如 "4321", "dcba"）
			isDescending := true
			for j := 1; j <= c.Maxsequence; j++ {
				if i+j >= len(runes) || runes[i+j] != runes[i+j-1]-1 {
					isDescending = false
					break
				}
			}
			if isAscending || isDescending {
				return errors.Wrapf(ErrPasswordTooWeak, "password contains a sequence of more than %d consecutive characters", c.Maxsequence)
			}
		}
	}

	// 检查密码中是否包含用户名
	if c.Usercheck > 0 && username != "" {
		// 将用户名和密码都转换为小写进行比较（不区分大小写）
		lowerUsername := strings.ToLower(username)
		lowerPassword := strings.ToLower(password)

		// 检查密码中是否包含用户名（包括反向）
		if strings.Contains(lowerPassword, lowerUsername) {
			return errors.Wrapf(ErrPasswordTooWeak, "password contains the username")
		}

		// 检查密码中是否包含用户名的反向（用户名长度至少为3才检查反向）
		if len(lowerUsername) >= 3 {
			reversedUsername := reverseString(lowerUsername)
			if strings.Contains(lowerPassword, reversedUsername) {
				return errors.Wrapf(ErrPasswordTooWeak, "password contains the reversed username")
			}
		}
	}

	return nil
}

// reverseString 反转字符串
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ParsePAMConfig 解析 PAM 配置文件中的密码强度策略
// 支持 pam_pwquality 和 pam_cracklib 模块
//
// 兼容性说明：
//   - enforcing 和 enforce_for_root 参数在 libpwquality 1.2.0+ 版本中支持
//   - 较老的系统（如 RHEL 6 之前）可能不支持这些参数
//   - 在 PAM 配置中，enforce_for_root 可能以独立标志形式出现（无值），如：
//     password requisite pam_pwquality.so minlen=8 enforce_for_root
//     这种情况下，如果解析到 enforce_for_root 标志（无值），将设置为 1
//   - 如果系统不支持这些参数，配置文件中不会出现，将使用默认值
func ParsePAMConfig(content []byte, config *Config) *Config {
	if config == nil {
		config = &Config{
			Minlen:         0,
			Dcredit:        0,
			Ucredit:        0,
			Lcredit:        0,
			Ocredit:        0,
			Minclass:       0,
			Maxrepeat:      0,
			Maxclassrepeat: 0,
			Maxsequence:    0,
			Enforcing:      1, // 默认值（1 表示强制执行）
			EnforceForRoot: 0, // 默认值（0 表示不对 root 强制执行）
			Usercheck:      0, // 默认值（0 表示不检查用户名）
		}
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过注释和空行
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// 查找 password 相关的 PAM 配置行
		// 格式: password requisite pam_pwquality.so retry=3 minlen=8 dcredit=-1 ucredit=-1
		// 或: password requisite pam_cracklib.so retry=3 minlen=8 dcredit=-1 ucredit=-1
		if !strings.Contains(line, "password") {
			continue
		}

		// 检查是否包含 pam_pwquality 或 pam_cracklib
		if !strings.Contains(line, "pam_pwquality") && !strings.Contains(line, "pam_cracklib") {
			continue
		}

		// 解析参数，格式为 key=value 或独立标志（如 enforce_for_root）
		// 先找到 .so 后面的参数部分
		parts := strings.Fields(line)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) == 0 {
				continue
			}

			// 检查是否是 key=value 格式
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) != 2 {
					continue
				}

				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])

				switch key {
				case "minlen":
					if v, err := strconv.Atoi(value); err == nil {
						config.Minlen = v
					}
				case "dcredit":
					if v, err := strconv.Atoi(value); err == nil {
						config.Dcredit = v
					}
				case "ucredit":
					if v, err := strconv.Atoi(value); err == nil {
						config.Ucredit = v
					}
				case "lcredit":
					if v, err := strconv.Atoi(value); err == nil {
						config.Lcredit = v
					}
				case "ocredit":
					if v, err := strconv.Atoi(value); err == nil {
						config.Ocredit = v
					}
				case "minclass":
					if v, err := strconv.Atoi(value); err == nil {
						config.Minclass = v
					}
				case "maxrepeat":
					if v, err := strconv.Atoi(value); err == nil {
						config.Maxrepeat = v
					}
				case "maxclassrepeat":
					if v, err := strconv.Atoi(value); err == nil {
						config.Maxclassrepeat = v
					}
				case "maxsequence":
					if v, err := strconv.Atoi(value); err == nil {
						config.Maxsequence = v
					}
				case "enforcing":
					if v, err := strconv.Atoi(value); err == nil {
						config.Enforcing = v
					}
				case "enforce_for_root":
					if v, err := strconv.Atoi(value); err == nil {
						config.EnforceForRoot = v
					}
				case "usercheck":
					if v, err := strconv.Atoi(value); err == nil {
						config.Usercheck = v
					}
				case "difok": // pam_cracklib 特有：至少需要多少个字符与旧密码不同
					// 这个参数不影响密码强度校验，可以忽略
				case "retry": // 重试次数，不影响密码强度校验
				}
			} else {
				// 处理独立标志（无值的参数）
				// 例如：enforce_for_root（表示对 root 强制执行密码策略）
				switch part {
				case "enforce_for_root":
					// 如果以独立标志形式出现，设置为 1（强制执行）
					config.EnforceForRoot = 1
				}
			}
		}
	}

	return config
}

// GeneratePassword 根据配置生成符合强度要求的密码
// passwordGenerator 是一个函数，接受长度参数并返回密码
// 如果 passwordGenerator 为 nil，将使用默认的最小长度 12
func (c *Config) GeneratePassword(passwordGenerator func(int) string) string {
	if c == nil || !c.HasAnyPolicy() {
		// 如果没有配置或配置为空，使用默认长度生成
		if passwordGenerator != nil {
			return passwordGenerator(12)
		}
		return ""
	}

	// 计算所需的最小密码长度
	minLength := c.Minlen
	if minLength == 0 {
		minLength = 8 // 默认最小长度
	}

	// 根据 credit 要求计算额外需要的字符数
	requiredChars := 0
	if c.Dcredit < 0 {
		requiredChars += -c.Dcredit
	}
	if c.Ucredit < 0 {
		requiredChars += -c.Ucredit
	}
	if c.Lcredit < 0 {
		requiredChars += -c.Lcredit
	}
	if c.Ocredit < 0 {
		requiredChars += -c.Ocredit
	}

	// 确保长度满足所有要求
	passwordLength := minLength
	if requiredChars > 0 {
		// 至少需要 minLength 和 requiredChars 中的较大值
		if requiredChars > passwordLength {
			passwordLength = requiredChars
		}
		// 再加上一些缓冲，确保有足够的字符满足 minclass 要求
		if c.Minclass > 0 && c.Minclass > 1 {
			passwordLength += c.Minclass
		}
	}

	if passwordGenerator == nil {
		return ""
	}

	// 生成密码并验证，直到符合要求
	maxAttempts := 64
	for i := 12; i < maxAttempts; i += 2 {
		password := passwordGenerator(passwordLength)
		// GeneratePassword 不提供用户名，所以传空字符串
		if c.Validate(password, "") == nil {
			return password
		}
		// 如果不符合要求，增加长度重试
		passwordLength++
	}

	// 如果多次尝试都失败，返回一个较长的密码（应该能满足大部分要求）
	return passwordGenerator(passwordLength)
}
