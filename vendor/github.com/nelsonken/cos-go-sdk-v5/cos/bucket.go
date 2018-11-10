package cos

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Bucket bucket
type Bucket struct {
	Name string
	conn *Conn
}

// ObjectSlice object slice
type ObjectSlice struct {
	UploadID string
	Size     int64
	Offset   int64
	Number   int
	MD5      string
	Dst      string
	Result   bool
}

// 获得云存储上文件信息
func (b *Bucket) HeadObject(ctx context.Context, object string) error {
	resq, err := b.conn.Do(ctx, http.MethodHead, b.Name, object, nil, nil, nil)
	if err == nil {
		defer resq.Body.Close()
	} else {
		for k, v := range resq.Header {
			value := fmt.Sprintf("%s", v)
			fmt.Printf("%-18s: %s\n", k, strings.Replace(strings.Replace(value, "[", "", -1), "]", "", -1))
		}
	}

	return err
}

func (b *Bucket) UploadObject(ctx context.Context, object string, content io.Reader, acl *AccessControl) error {
	res, err := b.conn.Do(ctx, http.MethodPut, b.Name, object, nil, acl.GenHead(), content)
	if err == nil {
		defer res.Body.Close()
	}

	return err
}

func (b *Bucket) CopyObject(ctx context.Context, src, dst string, acl *AccessControl) error {
	srcURL := fmt.Sprintf("%s-%s.cos.%s.%s/%s", b.Name, b.conn.conf.AppID, b.conn.conf.Region, b.conn.conf.Domain, dst)
	header := map[string]string{
		"x-cos-source-url": srcURL,
	}

	res, err := b.conn.Do(ctx, http.MethodPut, b.Name, dst, nil, header, nil)
	if err == nil {
		defer res.Body.Close()
	}

	return err
}

func (b *Bucket) DeleteObject(ctx context.Context, obj string) error {
	res, err := b.conn.Do(ctx, http.MethodDelete, b.Name, obj, nil, nil, nil)
	if err == nil {
		defer res.Body.Close()
	}

	return err
}

func (b *Bucket) DownloadObject(ctx context.Context, object string, w io.Writer) error {
	res, err := b.conn.Do(ctx, http.MethodGet, b.Name, object, nil, nil, nil)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, res.Body)

	return err
}

// UploadObjectBySlice upload by slice
func (b *Bucket) UploadObjectBySlice(ctx context.Context, dst, src string, taskNum int, headers map[string]string) error {
	if taskNum < 1 {
		return ParamError{"taskNum 必须大于1"}
	}

	uploadID, err := b.InitSliceUpload(ctx, dst, headers)
	if err != nil {
		return err
	}

	fd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fd.Close()

	slices, err := b.PerformSliceUpload(ctx, dst, uploadID, fd, taskNum)
	if err != nil {
		return err
	}

	err = b.CompleteSliceUpload(ctx, dst, uploadID, fd, slices)

	return err
}

// InitSliceUpload init upload by slice
func (b *Bucket) InitSliceUpload(ctx context.Context, obj string, headers map[string]string) (string, error) {
	param := map[string]interface{}{
		"uploads": "",
	}
	res, err := b.conn.Do(ctx, http.MethodPost, b.Name, obj, param, headers, nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	imur := &InitiateMultipartUploadResult{}
	err = XMLDecode(res.Body, imur)
	if err != nil {
		return "", err
	}

	return imur.UploadID, nil
}

// CompleteSliceUpload finish slice Upload
func (b *Bucket) CompleteSliceUpload(ctx context.Context, dst, uploadID string, fd *os.File, slice []*ObjectSlice) error {
	cmu := &CompleteMultipartUpload{}
	cmu.Part = []struct {
		PartNumber int
		ETag       string
	}{}

	for _, osl := range slice {
		cmu.Part = append(cmu.Part, struct {
			PartNumber int
			ETag       string
		}{PartNumber: osl.Number, ETag: osl.MD5})
	}

	cmuXML, err := xml.Marshal(cmu)
	if err != nil {
		return err
	}
	param := map[string]interface{}{
		"uploadId": uploadID,
	}
	res, err := b.conn.Do(ctx, http.MethodPost, b.Name, dst, param, nil, bytes.NewReader(cmuXML))
	if err == nil {
		defer res.Body.Close()
	}

	return err
}

// PerformSliceUpload perform slice upload
func (b *Bucket) PerformSliceUpload(ctx context.Context, dst, uploadID string, fd *os.File, taskNum int) ([]*ObjectSlice, error) {
	oss, err := b.getFileSlices(fd, uploadID, dst)
	if err != nil {
		return nil, err
	}

	jobNum := len(oss)
	jobs := make(chan *ObjectSlice, jobNum)
	result := make(chan *ObjectSlice, jobNum)

	for i := 0; i < taskNum; i++ {
		go b.Worker(ctx, fd, jobs, result)
	}

	for _, osl := range oss {
		jobs <- osl
	}
	close(jobs)

	for i := 0; i < jobNum; i++ {
		res := <-result
		if !res.Result {
			return nil, SliceError{fmt.Sprintf("part info : num:%d, md5:%s", res.Number, res.MD5)}
		}
	}

	return oss, nil
}

// Worker woker for slice upload
func (b *Bucket) Worker(ctx context.Context, fd *os.File, jobs <-chan *ObjectSlice, result chan<- *ObjectSlice) {
	for job := range jobs {
		content, err := getFilePartContent(fd, job.Offset, job.Size)
		if err != nil {
			continue
		}

		err = b.UploadSlice(ctx, job.UploadID, job.Dst, job.Number, job.MD5, content)
		if err == nil {
			job.Result = true
		} else {
			job.Result = false
		}

		result <- job
	}
}

// UploadSlice upload one slice
func (b *Bucket) UploadSlice(ctx context.Context, uploadID, dst string, number int, etag string, content io.Reader) error {
	param := map[string]interface{}{
		"PartNumber": number,
		"uploadId":   uploadID,
	}
	res, err := b.conn.Do(ctx, http.MethodPut, b.Name, dst, param, nil, content)

	if err != nil {
		return FileError{"PUT数据错误:" + err.Error()}
	}
	defer res.Body.Close()

	if strings.Trim(res.Header.Get("Etag"), "\"") != etag {
		return FileError{"cos-etag与文件MD5不匹配"}
	}

	return err
}

func (b *Bucket) getFileSlices(fd *os.File, uploadID, dst string) ([]*ObjectSlice, error) {
	sliceSize := b.conn.conf.PartSize
	fi, err := fd.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fi.Size()
	oss := []*ObjectSlice{}
	var i int
	var offset int64
	for fileSize > 0 {
		var size int64
		if fileSize > sliceSize {
			size = sliceSize
		} else {
			size = fileSize
		}
		i++
		md5, err := getFileMD5(fd, offset, size)
		if err != nil {
			return nil, err
		}

		osl := &ObjectSlice{}
		osl.Size = size
		osl.Number = i
		osl.Offset = offset
		osl.UploadID = uploadID
		osl.MD5 = md5
		osl.Dst = dst
		oss = append(oss, osl)

		fileSize -= sliceSize
		offset += sliceSize
	}

	return oss, nil
}

func getFileMD5(fd *os.File, offset, size int64) (string, error) {
	buf := make([]byte, size)
	_, err := fd.ReadAt(buf, offset)
	if err != nil {
		return "", err
	}

	encoder := md5.New()
	encoder.Write(buf)
	b := encoder.Sum(nil)

	return hex.EncodeToString(b), nil
}

func getFilePartContent(fd *os.File, offset, size int64) (io.Reader, error) {
	buf := make([]byte, size)
	_, err := fd.ReadAt(buf, offset)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), nil
}

func (b *Bucket) AbortUpload(ctx context.Context, obj, uploadID string) error {
	param := map[string]interface{}{
		"uploadId": uploadID,
	}
	_, err := b.conn.Do(ctx, http.MethodDelete, b.Name, obj, param, nil, nil)

	return err
}

// ObjectExists object exists
func (b *Bucket) ObjectExists(ctx context.Context, obj string) error {
	_, err := b.conn.Do(ctx, http.MethodHead, b.Name, obj, nil, nil, nil)

	return err
}
