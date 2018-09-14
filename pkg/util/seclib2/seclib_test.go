package seclib2

import (
	"testing"
	"math/rand"
	"time"
)

func TestRandomPassword2(t *testing.T) {
	rand.Seed(time.Now().Unix())
	t.Logf("%s", RandomPassword2(12))
}
