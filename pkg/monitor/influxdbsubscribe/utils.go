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

package influxdbsubscribe

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var (
	Codes = map[byte][]byte{
		',': []byte(`\,`),
		'"': []byte(`\"`),
		' ': []byte(`\ `),
		'=': []byte(`\=`),
	}

	codesStr = map[string]string{}
)

func init() {
	for k, v := range Codes {
		codesStr[string(k)] = string(v)
	}
}

const (
	// MinNanoTime is the minumum time that can be represented.
	//
	// 1677-09-21 00:12:43.145224194 +0000 UTC
	//
	// The two lowest minimum integers are used as sentinel values.  The
	// minimum value needs to be used as a value lower than any other value for
	// comparisons and another separate value is needed to act as a sentinel
	// default value that is unusable by the user, but usable internally.
	// Because these two values need to be used for a special purpose, we do
	// not allow users to write points at these two times.
	MinNanoTime = int64(math.MinInt64) + 2

	// MaxNanoTime is the maximum time that can be represented.
	//
	// 2262-04-11 23:47:16.854775806 +0000 UTC
	//
	// The highest time represented by a nanosecond needs to be used for an
	// exclusive range in the shard group, so the maximum time needs to be one
	// less than the possible maximum number of nanoseconds representable by an
	// int64 so that we don't lose a point at that one time.
	MaxNanoTime = int64(math.MaxInt64) - 1
)

var (
	minNanoTime = time.Unix(0, MinNanoTime).UTC()
	maxNanoTime = time.Unix(0, MaxNanoTime).UTC()

	// ErrTimeOutOfRange gets returned when time is out of the representable range using int64 nanoseconds since the epoch.
	ErrTimeOutOfRange = fmt.Errorf("time outside range %d - %d", MinNanoTime, MaxNanoTime)
)

// parseIntBytes is a zero-alloc wrapper around strconv.ParseInt.
func parseIntBytes(b []byte, base int, bitSize int) (i int64, err error) {
	s := unsafeBytesToString(b)
	return strconv.ParseInt(s, base, bitSize)
}

// unsafeBytesToString converts a []byte to a string without a heap allocation.
//
// It is unsafe, and is intended to prepare input to short-lived functions
// that require strings.
func unsafeBytesToString(in []byte) string {
	src := *(*reflect.SliceHeader)(unsafe.Pointer(&in))
	dst := reflect.StringHeader{
		Data: src.Data,
		Len:  src.Len,
	}
	s := *(*string)(unsafe.Pointer(&dst))
	return s
}

// parseFloatBytes is a zero-alloc wrapper around strconv.ParseFloat.
func parseFloatBytes(b []byte, bitSize int) (float64, error) {
	s := unsafeBytesToString(b)
	return strconv.ParseFloat(s, bitSize)
}

// parseBoolBytes is a zero-alloc wrapper around strconv.ParseBool.
func parseBoolBytes(b []byte) (bool, error) {
	return strconv.ParseBool(unsafeBytesToString(b))
}

func Unescape(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}

	if bytes.IndexByte(in, '\\') == -1 {
		return in
	}

	i := 0
	inLen := len(in)
	var out []byte

	for {
		if i >= inLen {
			break
		}
		if in[i] == '\\' && i+1 < inLen {
			switch in[i+1] {
			case ',':
				out = append(out, ',')
				i += 2
				continue
			case '"':
				out = append(out, '"')
				i += 2
				continue
			case ' ':
				out = append(out, ' ')
				i += 2
				continue
			case '=':
				out = append(out, '=')
				i += 2
				continue
			}
		}
		out = append(out, in[i])
		i += 1
	}
	return out
}

func UnescapeString(in string) string {
	if strings.IndexByte(in, '\\') == -1 {
		return in
	}

	for b, esc := range codesStr {
		in = strings.Replace(in, esc, b, -1)
	}
	return in
}

func GetString(in string) string {
	for b, esc := range codesStr {
		in = strings.Replace(in, b, esc, -1)
	}
	return in
}

// SafeCalcTime safely calculates the time given. Will return error if the time is outside the
// supported range.
func SafeCalcTime(timestamp int64, precision string) (time.Time, error) {
	mult := GetPrecisionMultiplier(precision)
	if t, ok := safeSignedMult(timestamp, mult); ok {
		tme := time.Unix(0, t).UTC()
		return tme, CheckTime(tme)
	}

	return time.Time{}, ErrTimeOutOfRange
}

// CheckTime checks that a time is within the safe range.
func CheckTime(t time.Time) error {
	if t.Before(minNanoTime) || t.After(maxNanoTime) {
		return ErrTimeOutOfRange
	}
	return nil
}

// Perform the multiplication and check to make sure it didn't overflow.
func safeSignedMult(a, b int64) (int64, bool) {
	if a == 0 || b == 0 || a == 1 || b == 1 {
		return a * b, true
	}
	if a == MinNanoTime || b == MaxNanoTime {
		return 0, false
	}
	c := a * b
	return c, c/b == a
}

const escapeChars = `," =`

func IsEscaped(b []byte) bool {
	for len(b) > 0 {
		i := bytes.IndexByte(b, '\\')
		if i < 0 {
			return false
		}

		if i+1 < len(b) && strings.IndexByte(escapeChars, b[i+1]) >= 0 {
			return true
		}
		b = b[i+1:]
	}
	return false
}

func AppendUnescaped(dst, src []byte) []byte {
	var pos int
	for len(src) > 0 {
		next := bytes.IndexByte(src[pos:], '\\')
		if next < 0 || pos+next+1 >= len(src) {
			return append(dst, src...)
		}

		if pos+next+1 < len(src) && strings.IndexByte(escapeChars, src[pos+next+1]) >= 0 {
			if pos+next > 0 {
				dst = append(dst, src[:pos+next]...)
			}
			src = src[pos+next+1:]
			pos = 0
		} else {
			pos += next + 1
		}
	}

	return dst
}
