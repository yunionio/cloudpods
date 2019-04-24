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

package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"os"
	"strconv"
	"time"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <file> [sample]\n", os.Args[0])
		return
	}
	names := []string{"MD5", "SHA1", "SHA256", "SHA512"}
	hashes := []hash.Hash{md5.New(), sha1.New(), sha256.New(), sha512.New()}

	start := time.Now()

	var sample int
	var err error
	if len(os.Args) > 2 {
		sample, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("illegal sample %s", err)
			return
		}
	}

	var results [][]byte

	if sample > 0 {
		results, err = fileutils2.FileFastHash(os.Args[1], hashes, sample)
	} else {
		results, err = fileutils2.FileHash(os.Args[1], hashes)
	}
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}
	for i := 0; i < len(results); i += 1 {
		fmt.Printf("%s: %x\n", names[i], results[i])
	}
	sum := fileutils2.SumHashes(results)
	fmt.Printf("SUM: %x\n", sum)
	cost := time.Now().Sub(start)
	fmt.Printf("Time: %dms\n", cost/time.Millisecond)
}
