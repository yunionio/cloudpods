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

package atexit

import (
	"testing"
)

func TestAtExit(t *testing.T) {
	t.Run("empty reason", func(t *testing.T) {
		func() {
			defer func() {
				val := recover()
				if val == nil {
					t.Errorf("should panic")
				}
			}()
			Register(ExitHandler{})
		}()
	})
	t.Run("empty func", func(t *testing.T) {
		func() {
			defer func() {
				val := recover()
				if val == nil {
					t.Errorf("should panic")
				}
			}()
			Register(ExitHandler{
				Reason: "have reason",
			})
		}()
	})
	t.Run("order & prio", func(t *testing.T) {
		verdict := ""
		handler := func(eh ExitHandler) { verdict += eh.Reason }
		Register(ExitHandler{
			Prio:   2,
			Reason: "2",
			Func:   handler,
		})
		Register(ExitHandler{
			Prio:   0,
			Reason: "0",
			Func:   handler,
		})
		Register(ExitHandler{
			Prio:   0,
			Reason: "1",
			Func:   handler,
		})
		Handle()
		Handle()
		if verdict != "012" {
			t.Errorf("expecting %q, got %q", "012", verdict)
		}
	})
}
