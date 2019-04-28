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
