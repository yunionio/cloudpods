package httperrors

import (
	"fmt"
	"strings"
	"testing"
)

func TestGeneralError(t *testing.T) {
	t.Run("unclassified", func(t *testing.T) {
		t.Run("no fmt", func(t *testing.T) {
			err := fmt.Errorf("i am an unclassified error")
			jce := NewGeneralError(err)
			if strings.Contains(jce.Details, "%!(EXTRA") {
				t.Errorf("bad error formating: %v", jce)
			}
		})
		t.Run("fmt", func(t *testing.T) {
			err := fmt.Errorf("i am error with plain %%s")
			jce := NewGeneralError(err)
			if strings.Contains(jce.Details, "%!(EXTRA") {
				t.Errorf("bad error formating: %v", jce)
			}
		})
	})
}
