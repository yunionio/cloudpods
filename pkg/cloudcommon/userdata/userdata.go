// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package userdata

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/cloudinit"
	"yunion.io/x/pkg/util/osprofile"

	"yunion.io/x/onecloud/pkg/httperrors"
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

func ValidateUserdata(data string, osType string) error {
	if len(data) == 0 {
		return nil
	}
	_, err := cloudinit.ParseUserData(data)
	if err != nil {
		if osType == osprofile.OS_TYPE_WINDOWS {
			if strings.HasPrefix(data, "[bat]\n") || strings.HasPrefix(data, "[powershell]\n") {
				// valid
			} else {
				return errors.Wrap(httperrors.ErrInputParameter, "invalid windows scripts")
			}
		} else {
			if strings.HasPrefix(data, "#!/bin/sh\n") || strings.HasPrefix(data, "#!/bin/bash\n") || strings.HasPrefix(data, "#!/usr/bin/env bash") {
				// valid
			} else {
				return errors.Wrap(httperrors.ErrInputParameter, "invalid shell scripts")
			}
		}
	}
	encodeData, err := Encode(data)
	if err != nil {
		return errors.Wrapf(httperrors.ErrInputParameter, "Encode data error %s", err)
	}
	if len(encodeData) > UserdataLimitSize {
		return errors.Wrapf(httperrors.ErrInputParameter, "user data size %d large limit %d", len(encodeData), UserdataLimitSize)
	}
	return nil
}
