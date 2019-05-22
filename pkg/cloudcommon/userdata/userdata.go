package userdata

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
)

const (
	UserdataLimitSize = 64 * 1024 // 64 KB
)

func Encode(userdata string) (string, error) {
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(userdata)); err != nil {
		return "", errors.Wrap(err, "failed to gzip userdata")
	}
	if err := gz.Close(); err != nil {
		return "", errors.Wrap(err, "close gzip")
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func Decode(encodeUserdata string) (string, error) {
	gzData, err := base64.StdEncoding.DecodeString(encodeUserdata)
	if err != nil {
		return "", errors.Wrap(err, "base64 decode gzip data")
	}
	gr, err := gzip.NewReader(bytes.NewBuffer(gzData))
	if err != nil {
		return "", errors.Wrap(err, "new reader")
	}
	defer gr.Close()
	data, err := ioutil.ReadAll(gr)
	if err != nil {
		return "", errors.Wrap(err, "read data")
	}
	return string(data), nil
}

func ValidateUserdata(data string) error {
	if len(data) == 0 {
		return nil
	}
	encodeData, err := Encode(data)
	if err != nil {
		return errors.Wrapf(err, "Encode data")
	}
	if len(encodeData) > UserdataLimitSize {
		return errors.New(fmt.Sprintf("user data size %d large limit %d", len(encodeData), UserdataLimitSize))
	}
	return nil
}
