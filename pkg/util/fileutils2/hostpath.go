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

package fileutils2

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
)

var unixFileModeRe = regexp.MustCompile(`^[0-7]{3,4}$`)

var restrictedHostBindPrefixes = []string{
	"/etc",
	"/root",
	"/proc",
	"/sys",
	"/boot",
	"/dev",
	"/run",
	"/var/run",
	"/var/lib",
	"/usr",
	"/bin",
	"/sbin",
	"/lib",
	"/lib64",
}

// CleanHostBindPath returns a cleaned absolute host path suitable for bind-mount.
// The root filesystem path "/" is rejected.
func CleanHostBindPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.Errorf("path is empty")
	}
	if !filepath.IsAbs(p) {
		return "", errors.Errorf("path %q must be absolute", p)
	}
	clean := filepath.Clean(p)
	if !filepath.IsAbs(clean) {
		return "", errors.Errorf("path %q is not absolute after cleaning", p)
	}
	if clean == string(filepath.Separator) {
		return "", errors.Errorf("path %q is not allowed", p)
	}
	return clean, nil
}

// IsRestrictedHostBindPath reports whether a cleaned absolute path is under a
// host location that ordinary users must not bind into a container.
func IsRestrictedHostBindPath(p string) bool {
	p = filepath.Clean(p)
	for _, prefix := range restrictedHostBindPrefixes {
		if p == prefix || strings.HasPrefix(p, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// ParseUnixFileMode parses a chmod-style octal mode (e.g. "755" or "0755").
// The returned string is the canonical octal form without a leading zero.
func ParseUnixFileMode(perm string) (os.FileMode, string, error) {
	perm = strings.TrimSpace(perm)
	if perm == "" {
		return 0, "", errors.Errorf("file mode is empty")
	}
	if !unixFileModeRe.MatchString(perm) {
		return 0, "", errors.Errorf("invalid file mode %q", perm)
	}
	v, err := strconv.ParseUint(perm, 8, 32)
	if err != nil {
		return 0, "", errors.Wrapf(err, "invalid file mode %q", perm)
	}
	if v > 0o777 {
		return 0, "", errors.Errorf("invalid file mode %q", perm)
	}
	return os.FileMode(v), strconv.FormatUint(v, 8), nil
}

// CleanRelSubpath cleans a relative path and rejects absolute paths or parent
// directory traversal.
func CleanRelSubpath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.Errorf("path is empty")
	}
	if strings.ContainsRune(p, 0) {
		return "", errors.Errorf("path contains NUL")
	}
	if filepath.IsAbs(p) {
		return "", errors.Errorf("path %q must be relative", p)
	}
	clean := filepath.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.Errorf("path %q is not allowed", p)
	}
	if filepath.IsAbs(clean) {
		return "", errors.Errorf("path %q must be relative", p)
	}
	return clean, nil
}

// JoinInside joins elem onto base and requires the result to stay inside base.
func JoinInside(base, elem string) (string, error) {
	base = filepath.Clean(base)
	if elem == "" {
		return base, nil
	}
	rel, err := CleanRelSubpath(elem)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(base, rel)
	if !IsPathInside(base, joined) {
		return "", errors.Errorf("path %q escapes %q", elem, base)
	}
	return joined, nil
}

// JoinInsideAll joins elems onto base in order and requires each step to stay
// inside the previous result.
func JoinInsideAll(base string, elems ...string) (string, error) {
	cur := filepath.Clean(base)
	for _, e := range elems {
		if e == "" {
			continue
		}
		next, err := JoinInside(cur, e)
		if err != nil {
			return "", err
		}
		cur = next
	}
	return cur, nil
}
