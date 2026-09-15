// Package platform handles the parsing and comparing of the image platform (e.g. linux/amd64)
package platform

// Some of the code in the package and all of the inspiration for this comes from <https://github.com/containerd/containerd>.
// Their license is included here:
/*
   Copyright The containerd Authors.
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
       http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"fmt"
	"path"
	"regexp"
	"runtime"
	"strings"
)

var (
	partRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	verRE  = regexp.MustCompile(`^[A-Za-z0-9\._-]+$`)
)

// Platform specifies a platform where a particular image manifest is applicable.
type Platform struct {
	// Architecture field specifies the CPU architecture, for example `amd64` or `ppc64`.
	Architecture string `json:"architecture"`

	// OS specifies the operating system, for example `linux` or `windows`.
	OS string `json:"os"`

	// OSVersion is an optional field specifying the operating system version, for example `10.0.10586`.
	OSVersion string `json:"os.version,omitempty"`

	// OSFeatures is an optional field specifying an array of strings, each listing a required OS feature (for example on Windows `win32k`).
	OSFeatures []string `json:"os.features,omitempty"`

	// Variant is an optional field specifying a variant of the CPU, for example `ppc64le` to specify a little-endian version of a PowerPC CPU.
	Variant string `json:"variant,omitempty"`

	// Features is an optional field specifying an array of strings, each listing a required CPU feature (for example `sse4` or `aes`).
	Features []string `json:"features,omitempty"`
}

// String outputs the platform in the <os>/<arch>/<variant> notation
func (p Platform) String() string {
	(&p).normalize()
	if p.OS == "" {
		return "unknown"
	} else if p.OS == "windows" {
		return path.Join(p.OS, p.Architecture, p.OSVersion)
	} else {
		return path.Join(p.OS, p.Architecture, p.Variant)
	}
}

// Compatible indicates if a host can run a specified target platform image.
// This accounts for Docker Desktop for Mac and Windows using a Linux VM.
func Compatible(host, target Platform) bool {
	(&host).normalize()
	(&target).normalize()
	if host.OS == "linux" {
		return host.OS == target.OS && host.Architecture == target.Architecture && host.Variant == target.Variant
	} else if host.OS == "windows" {
		if target.OS == "windows" {
			return host.Architecture == target.Architecture && host.Variant == target.Variant &&
				prefix(host.OSVersion) == prefix(target.OSVersion)
		} else if target.OS == "linux" {
			return host.Architecture == target.Architecture && host.Variant == target.Variant
		}
		return false
	} else if host.OS == "darwin" {
		if target.OS == "darwin" || target.OS == "linux" {
			return host.Architecture == target.Architecture && host.Variant == target.Variant
		}
		return false
	} else {
		return host.Architecture == target.Architecture &&
			host.OSVersion == target.OSVersion &&
			strSliceEq(host.OSFeatures, target.OSFeatures) &&
			host.Variant == target.Variant &&
			strSliceEq(host.Features, target.Features)
	}
}

// Match indicates if two platforms are the same
func Match(a, b Platform) bool {
	(&a).normalize()
	(&b).normalize()
	if a.OS != b.OS {
		return false
	}
	if a.OS == "linux" {
		return a.Architecture == b.Architecture && a.Variant == b.Variant
	} else if a.OS == "windows" {
		return a.Architecture == b.Architecture &&
			prefix(a.OSVersion) == prefix(b.OSVersion)
	} else {
		return a.Architecture == b.Architecture &&
			a.OSVersion == b.OSVersion &&
			strSliceEq(a.OSFeatures, b.OSFeatures) &&
			a.Variant == b.Variant &&
			strSliceEq(a.Features, b.Features)
	}
}

// Parse converts a platform string into a struct
func Parse(platStr string) (Platform, error) {
	// split on slash, validate each component
	platSplit := strings.Split(platStr, "/")
	for i, part := range platSplit {
		if i == 2 && platSplit[0] == "windows" {
			// TODO: (bmitch) this may not be officially allowed, but I can't find a decent reference for what it should be
			if !verRE.MatchString(part) {
				return Platform{}, fmt.Errorf("invalid platform component %s in %s", part, platStr)
			}
		} else if !partRE.MatchString(part) {
			return Platform{}, fmt.Errorf("invalid platform component %s in %s", part, platStr)
		}
		platSplit[i] = strings.ToLower(part)
	}
	plat := &Platform{}
	if len(platSplit) >= 1 {
		plat.OS = platSplit[0]
	}
	if len(platSplit) >= 2 {
		plat.Architecture = platSplit[1]
	}
	if len(platSplit) >= 3 {
		if plat.OS == "windows" {
			plat.OSVersion = platSplit[2]
		} else {
			plat.Variant = platSplit[2]
		}
	}
	// extrapolate missing fields and normalize
	platLocal := Local()
	if plat.OS == "" || plat.OS == "local" {
		// assume local OS
		plat.OS = platLocal.OS
	}
	switch plat.OS {
	case "macos":
		plat.OS = "darwin"
	}
	if len(platSplit) < 2 && plat.OS == runtime.GOOS {
		switch plat.OS {
		case "linux", "darwin":
			// automatically expand local architecture with recognized OS
			plat.Architecture = platLocal.Architecture
		case "windows":
			plat.Architecture = platLocal.Architecture
			plat.OSVersion = platLocal.OSVersion
		}
	}
	plat.normalize()

	return *plat, nil
}

func (p *Platform) normalize() {
	switch p.Architecture {
	case "i386":
		p.Architecture = "386"
		p.Variant = ""
	case "x86_64", "x86-64", "amd64":
		p.Architecture = "amd64"
		if p.Variant == "v1" {
			p.Variant = ""
		}
	case "aarch64", "arm64":
		p.Architecture = "arm64"
		switch p.Variant {
		case "8", "v8":
			p.Variant = ""
		}
	case "armhf":
		p.Architecture = "arm"
		p.Variant = "v7"
	case "armel":
		p.Architecture = "arm"
		p.Variant = "v6"
	case "arm":
		switch p.Variant {
		case "", "7":
			p.Variant = "v7"
		case "5", "6", "8":
			p.Variant = "v" + p.Variant
		}
	}
}

func prefix(platVer string) string {
	verParts := strings.Split(platVer, ".")
	if len(verParts) < 4 {
		return platVer
	}
	return strings.Join(verParts[0:3], ".")
}

func strSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
