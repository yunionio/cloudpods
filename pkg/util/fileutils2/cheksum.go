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

package fileutils2

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"

	"yunion.io/x/log"
)

func FileHash(filename string, hash []hash.Hash) ([][]byte, error) {
	fp, err := os.Open(filename)
	if err != nil {
		log.Errorf("open file for hash fail %s", err)
		return nil, err
	}
	defer fp.Close()

	buf := make([]byte, 4096)
	for {
		n, err := fp.Read(buf)
		if n > 0 {
			for i := 0; i < len(hash); i += 1 {
				hash[i].Write(buf[:n])
			}
		}
		if n == 0 || err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("read file error %s", err)
			return nil, err
		}
	}
	sums := make([][]byte, len(hash))
	for i := 0; i < len(hash); i += 1 {
		sums[i] = hash[i].Sum(nil)
	}
	return sums, nil
}

func MD5(filename string) (string, error) {
	sums, err := FileHash(filename, []hash.Hash{md5.New()})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sums[0]), nil
}

func SHA1(filename string) (string, error) {
	sums, err := FileHash(filename, []hash.Hash{sha1.New()})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sums[0]), nil
}

func SHA256(filename string) (string, error) {
	sums, err := FileHash(filename, []hash.Hash{sha256.New()})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sums[0]), nil
}

func SHA512(filename string) (string, error) {
	sums, err := FileHash(filename, []hash.Hash{sha512.New()})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sums[0]), nil
}
