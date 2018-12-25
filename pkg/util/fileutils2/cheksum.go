package fileutils2

import (
	"fmt"
	"hash"
	"io"
	"os"

	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
<<<<<<< HEAD

	"yunion.io/x/log"
)

func FileHash(filename string, hash []hash.Hash) ([][]byte, error) {
=======
	"yunion.io/x/log"
)

func FileHash(filename string, hash []hash.Hash) ([]string, error) {
>>>>>>> move util to pkg/utils
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
<<<<<<< HEAD
	sums := make([][]byte, len(hash))
	for i := 0; i < len(hash); i += 1 {
		sums[i] = hash[i].Sum(nil)
=======
	sums := make([]string, len(hash))
	for i := 0; i < len(hash); i += 1 {
		sums[i] = fmt.Sprintf("%x", hash[i].Sum(nil))
>>>>>>> move util to pkg/utils
	}
	return sums, nil
}

<<<<<<< HEAD
func MD5(filename string) (string, error) {
=======
func Md5(filename string) (string, error) {
>>>>>>> move util to pkg/utils
	sums, err := FileHash(filename, []hash.Hash{md5.New()})
	if err != nil {
		return "", err
	}
<<<<<<< HEAD
	return fmt.Sprintf("%x", sums[0]), nil
=======
	return sums[0], nil
>>>>>>> move util to pkg/utils
}

func SHA1(filename string) (string, error) {
	sums, err := FileHash(filename, []hash.Hash{sha1.New()})
	if err != nil {
		return "", err
	}
<<<<<<< HEAD
	return fmt.Sprintf("%x", sums[0]), nil
=======
	return sums[0], nil
>>>>>>> move util to pkg/utils
}

func SHA256(filename string) (string, error) {
	sums, err := FileHash(filename, []hash.Hash{sha256.New()})
	if err != nil {
		return "", err
	}
<<<<<<< HEAD
	return fmt.Sprintf("%x", sums[0]), nil
=======
	return sums[0], nil
>>>>>>> move util to pkg/utils
}

func SHA512(filename string) (string, error) {
	sums, err := FileHash(filename, []hash.Hash{sha512.New()})
	if err != nil {
		return "", err
	}
<<<<<<< HEAD
	return fmt.Sprintf("%x", sums[0]), nil
=======
	return sums[0], nil
>>>>>>> move util to pkg/utils
}
