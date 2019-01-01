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
