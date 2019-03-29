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

const (
	BLOCK_SIZE  = 32 * 1024 // 2**15
	BLOCK_WIDTH = 15
)

func SumHashes(sums [][]byte) []byte {
	minLen := 0
	for i := 0; i < len(sums); i += 1 {
		if minLen == 0 || minLen > len(sums[i]) {
			minLen = len(sums[i])
		}
	}
	ret := make([]byte, minLen)
	for j := 0; j < minLen; j += 1 {
		for i := 0; i < len(sums); i += 1 {
			ret[j] += sums[i][j]
		}
	}
	return ret
}

func FileFastHash(filename string, hashAlgo []hash.Hash, rate int) ([][]byte, error) {
	size := FileSize(filename)
	blockCount := size >> BLOCK_WIDTH
	samples := int(blockCount / int64(rate))

	// log.Infof("block_count: %d samples: %d", blockCount, samples)

	if samples == 0 {
		return FileHash(filename, hashAlgo)
	}

	fp, err := os.Open(filename)
	if err != nil {
		log.Errorf("open file for hash fail %s", err)
		return nil, err
	}
	defer fp.Close()

	buf := make([]byte, BLOCK_SIZE)
	offset := int64(0)
	for i := 0; i < samples; i += 1 {
		// log.Infof("%dth offset %d %d", i, offset, size)
		_, err := fp.Seek(offset, io.SeekStart)
		if err != nil {
			log.Errorf("seek error %s", err)
			return nil, err
		}
		n, err := fp.Read(buf)
		if err != nil {
			log.Errorf("read error %s", err)
			return nil, err
		}
		if n != BLOCK_SIZE {
			return nil, fmt.Errorf("fail to read all???")
		}
		for i := 0; i < len(hashAlgo); i += 1 {
			hashAlgo[i].Write(buf)
		}
		offset += (int64(rate) << BLOCK_WIDTH)
	}

	sums := make([][]byte, len(hashAlgo))
	for i := 0; i < len(hashAlgo); i += 1 {
		sums[i] = hashAlgo[i].Sum(nil)
	}
	return sums, nil
}

func FastCheckSum(filePath string) (string, error) {
	hashes := []hash.Hash{md5.New(), sha1.New(), sha256.New(), sha512.New()}
	results, err := FileFastHash(filePath, hashes, 128)
	if err != nil {
		return "", err
	}
	sum := SumHashes(results)
	return fmt.Sprintf("%x", sum), nil
}
