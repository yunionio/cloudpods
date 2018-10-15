package seclib2

import "testing"

func TestGeneratePassword(t *testing.T) {
	passwd := "Hello world!"
	dk, err := GeneratePassword(passwd)
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	t.Logf("%s", dk)

	err = VerifyPassword(passwd, dk)
	if err != nil {
		t.Errorf("fail to verify %s", err)
	}
}
