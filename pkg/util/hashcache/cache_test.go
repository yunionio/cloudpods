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

package hashcache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := NewCache(1024, time.Second)
	c.Set("123", 123)
	c.Set("456", 456)
	v := c.Get("123")
	if v == nil || v.(int) != 123 {
		t.Error("Key 123 not found")
	}
	v = c.Get("456")
	if v == nil || v.(int) != 456 {
		t.Error("Key 456 not found")
	}
	c.Set("456", 789)
	v = c.Get("456")
	if v == nil || v.(int) != 789 {
		t.Error("Key 456 not changed")
	}

	time.Sleep(time.Second)

	v = c.Get("123")
	if v != nil {
		t.Errorf("key 123 shoud expire")
	}
	c.Set("123", 1234)
	c.Set("456", 4567)
	v = c.Get("123")
	if v == nil || v.(int) != 1234 {
		t.Error("Key 123 not found")
	}
	v = c.Get("456")
	if v == nil || v.(int) != 4567 {
		t.Error("Key 456 not found")
	}

	time.Sleep(time.Second)

	v = c.Get("123")
	if v != nil {
		t.Errorf("key 123 shoud expire")
	}

	c.Set("123", 1234)
	v = c.Get("123")
	if v == nil || v.(int) != 1234 {
		t.Error("Key 123 not found")
	}
	c.Invalidate()
	v = c.Get("123")
	if v != nil {
		t.Error("Key 123 should not found")
	}
}
