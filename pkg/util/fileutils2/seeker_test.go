package fileutils2

import (
	"io"
	"strings"
	"testing"
)

func TestNewReadSeeker(t *testing.T) {
	testStr := "This is a test reader string"
	seeker, err := NewReadSeeker(strings.NewReader(testStr), int64(len(testStr)))
	if err != nil {
		t.Fatalf("NewReadSeeker error %s", err)
	}
	defer seeker.Close()
	buf1 := make([]byte, 1024)
	n, err := seeker.Read(buf1)
	if n != len(testStr) {
		t.Fatalf("read buf1 error %s", err)
	}
	buf2 := make([]byte, 1024)
	n, err = seeker.Read(buf2)
	if n != 0 {
		t.Fatalf("read buf2 should fail")
	}
	seeker.Seek(0, io.SeekStart)
	n, err = seeker.Read(buf2)
	if n != len(testStr) {
		t.Fatalf("read buf2 error %s", err)
	}
}
