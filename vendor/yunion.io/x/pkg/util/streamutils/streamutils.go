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

package streamutils

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
)

type SStreamProperty struct {
	CheckSum string
	Size     int64
}

func StreamPipe(reader io.Reader, writer io.Writer, CalChecksum bool, callback func(saved int64)) (*SStreamProperty, error) {
	sp := SStreamProperty{}

	var md5sum hash.Hash
	if CalChecksum {
		md5sum = md5.New()
	}

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			sp.Size += int64(n)
			if CalChecksum {
				md5sum.Write(buf[:n])
			}
			offset := 0
			for offset < n {
				m, err := writer.Write(buf[offset:n])
				if err != nil {
					return nil, err
				}
				offset += m
			}
			if callback != nil {
				callback(sp.Size)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	if CalChecksum {
		sp.CheckSum = fmt.Sprintf("%x", md5sum.Sum(nil))
	}
	return &sp, nil
}
