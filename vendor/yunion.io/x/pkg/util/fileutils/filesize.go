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

package fileutils

import (
	"fmt"
	"strconv"

	"yunion.io/x/pkg/util/regutils"
)

func parseSizeStr(sizeStr string, defaultUnit byte, base int) (int, error) {
	var numPart int
	var unit byte
	if regutils.MatchInteger(sizeStr) {
		numPart, _ = strconv.Atoi(sizeStr)
		unit = defaultUnit
	} else if regutils.MatchSize(sizeStr) {
		numPart, _ = strconv.Atoi(sizeStr[:len(sizeStr)-1])
		unit = sizeStr[len(sizeStr)-1]
	} else {
		return 0, fmt.Errorf("Invalid size string %s", sizeStr)
	}
	switch unit {
	case 'g', 'G':
		return numPart * base * base * base, nil
	case 'm', 'M':
		return numPart * base * base, nil
	case 'k', 'K':
		return numPart * base, nil
	case 't', 'T':
		return numPart * base * base * base * base, nil
	case 'p', 'P':
		return numPart * base * base * base * base * base, nil
	default:
		return numPart, nil
	}
}

func GetSizeGb(sizeStr string, defaultSize byte, base int) (int, error) {
	size, err := parseSizeStr(sizeStr, defaultSize, base)
	return size / base / base / base, err
}

func GetSizeMb(sizeStr string, defaultSize byte, base int) (int, error) {
	size, err := parseSizeStr(sizeStr, defaultSize, base)
	return size / base / base, err
}

func GetSizeKb(sizeStr string, defaultSize byte, base int) (int, error) {
	size, err := parseSizeStr(sizeStr, defaultSize, base)
	return size / base, err
}

func GetSizeBytes(sizeStr string, base int) (int, error) {
	return parseSizeStr(sizeStr, 'B', base)
}
