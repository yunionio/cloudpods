package appsrv

import (
	"testing"
)

func TestCache(t *testing.T) {
	c := NewCache(1024)
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
}
