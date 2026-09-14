package registry

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func DetectFileType(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "open file %q", filePath)
	}
	defer f.Close()

	buff := make([]byte, 512)
	// why 512 bytes ? see http://golang.org/pkg/net/http/#DetectContentType
	_, err = f.Read(buff)
	if err != nil {
		return "", errors.Wrapf(err, "read file %q header", filePath)
	}
	filetype := http.DetectContentType(buff)
	log.Infof("detect filetype %q", filetype)
	return filetype, nil
}

func IsGzipFile(filePath string) (bool, error) {
	fileType, err := DetectFileType(filePath)
	if err != nil {
		return false, errors.Wrap(err, "DetectFileType")
	}
	return fileType == "application/x-gzip", nil
}

func UnZip(filePath string, dstPath string) error {
	gzipFile, err := os.Open(filePath)
	if err != nil {
		return errors.Wrapf(err, "open file %q", filePath)
	}
	gr, err := gzip.NewReader(gzipFile)
	if err != nil {
		return errors.Wrapf(err, "create gzip reader")
	}
	defer gr.Close()

	dstW, err := os.Create(dstPath)
	if err != nil {
		return errors.Wrapf(err, "create target file %q", dstPath)
	}
	defer dstW.Close()

	_, err = io.Copy(dstW, gr)
	if err != nil {
		return errors.Wrapf(err, "decompress %q to %q", filePath, dstPath)
	}
	return nil
}
