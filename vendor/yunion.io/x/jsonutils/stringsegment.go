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

package jsonutils

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	numberReg = regexp.MustCompile(`^\d+$`)
)

type sTextNumber struct {
	text     string
	number   int64
	isNumber bool
}

func (tn sTextNumber) compare(tn2 sTextNumber) int {
	if tn.isNumber && !tn2.isNumber {
		return -1
	} else if !tn.isNumber && tn2.isNumber {
		return 1
	} else if tn.isNumber && tn2.isNumber {
		if tn.number < tn2.number {
			return -1
		} else if tn.number > tn2.number {
			return 1
		} else {
			return 0
		}
	} else {
		// !tn.isNumber && !tn2.isNumber
		if tn.text < tn2.text {
			return -1
		} else if tn.text > tn2.text {
			return 1
		} else {
			return 0
		}
	}
}

func (tn sTextNumber) String() string {
	if tn.isNumber {
		return strconv.FormatInt(tn.number, 10)
	} else {
		return tn.text
	}
}

func string2TextNumber(str string) sTextNumber {
	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return sTextNumber{text: str, isNumber: false}
	} else {
		return sTextNumber{number: num, isNumber: true}
	}
}

func string2Segments(str string) []sTextNumber {
	segs := strings.Split(str, ".")
	ret := make([]sTextNumber, len(segs))
	for i := range segs {
		ret[i] = string2TextNumber(segs[i])
	}
	return ret
}

func segments2string(segs []sTextNumber) string {
	segStrs := make([]string, len(segs))
	for i := range segs {
		segStrs[i] = segs[i].String()
	}
	return strings.Join(segStrs, ".")
}

type sStringSegments [][]sTextNumber

func (ss sStringSegments) Len() int      { return len(ss) }
func (ss sStringSegments) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
func (ss sStringSegments) Less(i, j int) bool {
	if len(ss[i]) < len(ss[j]) {
		return true
	} else if len(ss[i]) > len(ss[j]) {
		return false
	}
	for ii := range ss[i] {
		ret := ss[i][ii].compare(ss[j][ii])
		if ret < 0 {
			return true
		} else if ret > 0 {
			return false
		}
	}
	return false
}

func strings2stringSegments(strs []string) sStringSegments {
	ret := make([][]sTextNumber, len(strs))
	for i := range strs {
		ret[i] = string2Segments(strs[i])
	}
	return ret
}

func stringSegments2Strings(ss sStringSegments) []string {
	ret := make([]string, len(ss))
	for i := range ss {
		ret[i] = segments2string(ss[i])
	}
	return ret
}
