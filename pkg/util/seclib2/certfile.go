package seclib2

import (
	"fmt"
	"io/ioutil"
	"strings"
)

const (
	certBeginString = "BEGIN CERTIFICATE"
)

func MergeCaCertFiles(cafile string, certfile string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "cerfile.*.crt")
	if err != nil {
		return "", fmt.Errorf("fail to open tempfile for ca cerfile: %s", err)
	}
	defer tmpfile.Close()

	cont, err := ioutil.ReadFile(certfile)
	if err != nil {
		return "", fmt.Errorf("fail to read certfile %s", err)
	}
	offset := strings.Index(string(cont), certBeginString)
	if offset < 0 {
		return "", fmt.Errorf("invalid certfile, no BEGIN CERTIFICATE found")
	}
	for offset > 0 && cont[offset-1] == '-' {
		offset -= 1
	}
	tmpfile.Write(cont[offset:])
	cont, err = ioutil.ReadFile(cafile)
	if err != nil {
		return "", fmt.Errorf("fail to read cafile %s", err)
	}
	tmpfile.Write(cont)

	return tmpfile.Name(), nil
}
