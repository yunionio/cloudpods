package fernetool

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestFernetKeys(t *testing.T) {
	m := SFernetKeyManager{}
	err := m.InitKeys("", 2)
	if err != nil {
		t.Fatalf("fail to initkeys %s", err)
	}
	buf := make([]byte, 128)
	for i := 0; i < 10; i += 1 {
		msgLen, err := rand.Read(buf)
		if err != nil {
			t.Fatalf("rand.Read fail %s", err)
		}
		msg, err := m.Encrypt(buf[:msgLen])
		if err != nil {
			t.Fatalf("fail to encrypt %s", err)
		}
		omsg := m.Decrypt(msg, time.Hour)
		if len(omsg) != msgLen {
			t.Fatalf("descrupt fail %s", err)
		}
		for i := 0; i < msgLen; i += 1 {
			if omsg[i] != buf[i] {
				t.Fatalf("not identical message!!")
			}
		}
	}
}
