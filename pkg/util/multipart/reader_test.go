package multipart

import (
	"testing"
	"strings"
	"bytes"
	"io"
)

func TestReader(t *testing.T) {
	body := "this is the file body message!!!"
	r := NewReader(strings.NewReader(body), "", "test.txt")
	var buf bytes.Buffer
	io.Copy(&buf, r)
	t.Logf("%s", r.FormDataContentType())
	t.Logf("%s", buf.String())
}
