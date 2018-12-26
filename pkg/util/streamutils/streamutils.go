package streamutils

import (
	"crypto/md5"
	"fmt"
	"io"
)

type SStreamProperty struct {
	CheckSum string
	Size     int64
}

func StreamPipe(reader io.Reader, writer io.Writer) (*SStreamProperty, error) {
	sp := SStreamProperty{}

	md5sum := md5.New()

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			sp.Size += int64(n)
			md5sum.Write(buf[:n])
			offset := 0
			for offset < n {
				m, err := writer.Write(buf[offset:n])
				if err != nil {
					return nil, err
				}
				offset += m
			}
		} else if n == 0 {
			break
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	sp.CheckSum = fmt.Sprintf("%x", md5sum.Sum(nil))
	return &sp, nil
}
