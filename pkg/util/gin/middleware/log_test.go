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

package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const checkMark = "\u2713"
const ballotX = "\u2717"

func Test_ErrorLogger(t *testing.T) {
	middleware := ErrorLogger()
	assert.NotNil(t, middleware, "Can't get ErrorLogger middleware")
}

func Test_Logger(t *testing.T) {
	middleware := Logger()
	assert.NotNil(t, middleware, "Can't get Logger middleware")
}

func Test_colorForMethod(t *testing.T) {
	colors := map[string]string{
		"GET":     blue,
		"POST":    cyan,
		"PUT":     yellow,
		"HEAD":    magenta,
		"OPTIONS": white,
		"UNKNOWN": reset}

	for method, color := range colors {
		expect := colorForMethod(method)
		assert.NotNil(t, expect, "Can't get color from method %s %s",
			method, ballotX)
		t.Logf("Check if color %s%s%s is the right one for this method",
			string(expect), method, reset)
		if assert.Equal(t, expect, color, "Method %s has NOT the right color %s",
			method, ballotX) {
			t.Logf("Method %s has the right color %s", method, checkMark)
		}
	}
}

func Test_colorForStatus(t *testing.T) {
	colors := map[int]string{
		200: green,
		301: white,
		404: yellow,
		500: red}

	for status, color := range colors {
		expect := colorForStatus(status)
		assert.NotNil(t, expect, "Can't get color from status %d %s",
			status, ballotX)
		t.Logf("Check if color %s%d%s is the right one for this status",
			string(expect), status, reset)
		if assert.Equal(t, expect, color, "Status %d has NOT the right color %s",
			status, ballotX) {
			t.Logf("Status %d has the right color %s", status, checkMark)
		}
	}
}
