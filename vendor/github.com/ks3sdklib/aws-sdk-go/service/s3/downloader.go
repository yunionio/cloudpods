package s3

import (
	"context"
	"errors"
	"fmt"
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/crc"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type DownloadFileInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Object key of the object.
	Key *string `location:"uri" locationName:"Key" type:"string" required:"true"`

	// The path of the file to be downloaded.
	DownloadFile *string `type:"string" locationName:"DownloadFile" required:"true"`

	// The size of each part.
	PartSize *int64 `type:"integer" locationName:"PartSize"`

	// The number of tasks to download the file.
	TaskNum *int64 `type:"integer" locationName:"TaskNum"`

	// Whether to enable checkpoint.
	EnableCheckpoint *bool `type:"boolean" locationName:"EnableCheckpoint"`

	// The directory to store the checkpoint file.
	CheckpointDir *string `type:"string" locationName:"CheckpointDir"`

	// The checkpoint file path.
	CheckpointFile *string `type:"string" locationName:"CheckpointFile"`

	// 下载的范围，range[0]为开始位置，range[1]为结束位置
	// range[0] 小于 0 时表示从文件头开始下载
	// range[1] 小于 0 时表示下载到文件末尾
	// range[0] 和 range[1] 都小于 0 时表示下载整个文件
	// range[0] 和 range[1] 都大于等于 0 时表示下载指定范围的文件
	// range[0] 大于 range[1] 且 range[1] 非负时表示下载整个文件
	// 例如：
	// range=[0, 99] 表示下载文件的前100个字节
	// range=[100, 199] 表示下载文件的第101个字节至第200个字节
	// range=[100, -1] 表示下载文件的第101个字节至文件末尾
	// range=[-1, 100] 表示下载文件的后100个字节
	// Downloads the specified range bytes of an object.
	Range []int64 `locationName:"Range" type:"list"`

	// Sets the Content-Type header of the response.
	ResponseContentType *string `location:"querystring" locationName:"response-content-type" type:"string"`

	// Sets the Content-Language header of the response.
	ResponseContentLanguage *string `location:"querystring" locationName:"response-content-language" type:"string"`

	// Sets the Expires header of the response.
	ResponseExpires *time.Time `location:"querystring" locationName:"response-expires" type:"timestamp" timestampFormat:"iso8601"`

	// Sets the Cache-Control header of the response.
	ResponseCacheControl *string `location:"querystring" locationName:"response-cache-control" type:"string"`

	// Sets the Content-Disposition header of the response
	ResponseContentDisposition *string `location:"querystring" locationName:"response-content-disposition" type:"string"`

	// Sets the Content-Encoding header of the response.
	ResponseContentEncoding *string `location:"querystring" locationName:"response-content-encoding" type:"string"`

	// Return the object only if it has been modified since the specified time,
	// otherwise return a 304 (not modified).
	IfModifiedSince *time.Time `location:"header" locationName:"If-Modified-Since" type:"timestamp" timestampFormat:"rfc822"`

	// Return the object only if it has not been modified since the specified time,
	// otherwise return a 412 (precondition failed).
	IfUnmodifiedSince *time.Time `location:"header" locationName:"If-Unmodified-Since" type:"timestamp" timestampFormat:"rfc822"`

	// Return the object only if its entity tag (ETag) is the same as the one specified,
	// otherwise return a 412 (precondition failed).
	IfMatch *string `location:"header" locationName:"If-Match" type:"string"`

	// Return the object only if its entity tag (ETag) is different from the one
	// specified, otherwise return a 304 (not modified).
	IfNoneMatch *string `location:"header" locationName:"If-None-Match" type:"string"`

	// Specify the encoding type of the client.
	// If you want to compress and transmit the returned content using gzip,
	// you need to add a request header: Accept-Encoding:gzip。
	// KS3 will determine whether to return gzip compressed data based on the
	// Content-Type and Object size (not less than 1 KB) of the object.
	// Value: gzip、br、deflate
	AcceptEncoding *string `location:"header" locationName:"Accept-Encoding" type:"string"`

	// Specifies the algorithm to use to when encrypting the object, eg: AES256.
	SSECustomerAlgorithm *string `location:"header" locationName:"x-amz-server-side-encryption-customer-algorithm" type:"string"`

	// Specifies the customer-provided encryption key for KS3 to use in encrypting data.
	SSECustomerKey *string `location:"header" locationName:"x-amz-server-side-encryption-customer-key" type:"string"`

	// Specifies the 128-bit MD5 digest of the encryption key according to RFC 1321.
	SSECustomerKeyMD5 *string `location:"header" locationName:"x-amz-server-side-encryption-customer-key-MD5" type:"string"`

	// Progress callback function
	ProgressFn aws.ProgressFunc `location:"function"`
}

type DownloadFileOutput struct {
	Bucket *string

	Key *string

	ETag *string

	ChecksumCRC64ECMA *string

	ObjectMeta map[string]*string
}

func (c *S3) DownloadFile(request *DownloadFileInput) (*DownloadFileOutput, error) {
	return c.DownloadFileWithContext(context.Background(), request)
}

func (c *S3) DownloadFileWithContext(ctx context.Context, request *DownloadFileInput) (*DownloadFileOutput, error) {
	return newDownloader(c, ctx, request).downloadFile()
}

type Downloader struct {
	client *S3

	context context.Context

	downloadFileRequest *DownloadFileInput

	downloadCheckpoint *DownloadCheckpoint

	CompletedSize int64

	downloadFileSize int64

	downloadFileMeta map[string]*string

	mu sync.Mutex

	error error
}

func newDownloader(s3 *S3, ctx context.Context, request *DownloadFileInput) *Downloader {
	return &Downloader{
		client:              s3,
		context:             ctx,
		downloadFileRequest: request,
	}
}

func (d *Downloader) downloadFile() (*DownloadFileOutput, error) {
	err := d.validate()
	if err != nil {
		return nil, err
	}

	d.downloadFileMeta, err = d.headObject()
	if err != nil {
		return nil, err
	}

	dcp, err := newDownloadCheckpoint(d)
	if err != nil {
		return nil, err
	}
	d.downloadCheckpoint = dcp

	if aws.ToBoolean(d.downloadFileRequest.EnableCheckpoint) {
		cpFilePath := aws.ToString(d.downloadFileRequest.CheckpointFile)
		if cpFilePath == "" {
			cpFilePath, err = generateDownloadCpFilePath(d.downloadFileRequest)
			if err != nil {
				return nil, err
			}
		}
		dcp.CpFilePath = cpFilePath

		err = dcp.load()
		if err != nil {
			return nil, err
		}

		if !FileExists(dcp.DownloadFilePath + TempFileSuffix) {
			dcp.PartETagList = make([]*CompletedPart, 0)
			dcp.remove()
		}
	}

	err = d.createDownloadDir(dcp.DownloadFilePath + TempFileSuffix)
	if err != nil {
		return nil, err
	}

	objectRange := d.getObjectRange()
	d.downloadFileSize = objectRange[1] - objectRange[0] + 1
	partSize := aws.ToLong(d.downloadFileRequest.PartSize)
	totalPartNum := (d.downloadFileSize-1)/partSize + 1
	tasks := make(chan DownloadPartTask, totalPartNum)

	var i int64
	for i = 0; i < totalPartNum; i++ {
		partNum := i + 1
		start := objectRange[0] + i*partSize
		end := Min(start+partSize-1, objectRange[1])
		actualPartSize := end - start + 1
		if d.getPartETag(partNum) != nil {
			d.publishProgress(actualPartSize)
		} else {
			downloadPartTask := DownloadPartTask{
				partNumber:     partNum,
				start:          start,
				end:            end,
				actualPartSize: actualPartSize,
			}
			tasks <- downloadPartTask
		}
	}
	close(tasks)

	var wg sync.WaitGroup
	for i = 0; i < aws.ToLong(d.downloadFileRequest.TaskNum); i++ {
		wg.Add(1)
		go d.runTask(tasks, &wg)
	}
	wg.Wait()

	if d.error != nil {
		return nil, d.error
	}

	if d.downloadFileRequest.Range == nil && d.client.Config.CrcCheckEnabled {
		clientCrc64 := d.getCrc64Ecma(dcp.PartETagList)
		serverCrc64, _ := strconv.ParseUint(aws.ToString(d.downloadFileMeta[HTTPHeaderAmzChecksumCrc64ecma]), 10, 64)
		d.client.Config.LogDebug("check file crc64, client crc64:%d, server crc64:%d", clientCrc64, serverCrc64)
		if serverCrc64 != 0 && clientCrc64 != serverCrc64 {
			return nil, errors.New(fmt.Sprintf("crc64 check failed, client crc64:%d, server crc64:%d", clientCrc64, serverCrc64))
		}
	}

	err = d.complete()
	if err != nil {
		return nil, err
	}

	return d.getDownloadFileOutput(), nil
}

func (d *Downloader) validate() error {
	request := d.downloadFileRequest
	if request == nil {
		return errors.New("download file request is required")
	}

	if aws.ToString(request.Bucket) == "" {
		return errors.New("bucket is required")
	}

	if aws.ToString(request.Key) == "" {
		return errors.New("key is required")
	}

	err := d.normalizeDownloadPath()
	if err != nil {
		return err
	}

	if request.PartSize == nil {
		request.PartSize = aws.Long(DefaultPartSize)
	} else if aws.ToLong(request.PartSize) < MinPartSize {
		request.PartSize = aws.Long(MinPartSize)
	} else if aws.ToLong(request.PartSize) > MaxPartSize {
		request.PartSize = aws.Long(MaxPartSize)
	}

	if aws.ToLong(request.TaskNum) <= 0 {
		request.TaskNum = aws.Long(DefaultTaskNum)
	}

	return nil
}

func (d *Downloader) getDownloadFileOutput() *DownloadFileOutput {
	return &DownloadFileOutput{
		Bucket:            d.downloadFileRequest.Bucket,
		Key:               d.downloadFileRequest.Key,
		ETag:              d.downloadFileMeta[HTTPHeaderEtag],
		ChecksumCRC64ECMA: d.downloadFileMeta[HTTPHeaderAmzChecksumCrc64ecma],
		ObjectMeta:        d.downloadFileMeta,
	}
}

func (d *Downloader) getActualPartSize(fileSize int64, partSize int64, partNum int64) int64 {
	offset := (partNum - 1) * partSize
	actualPartSize := partSize
	if offset+partSize >= fileSize {
		actualPartSize = fileSize - offset
	}
	return actualPartSize
}

func (d *Downloader) getPartETag(partNumber int64) *CompletedPart {
	for _, partETag := range d.downloadCheckpoint.PartETagList {
		if *partETag.PartNumber == partNumber {
			return partETag
		}
	}
	return nil
}

type DownloadPartTask struct {
	partNumber int64

	actualPartSize int64

	start int64

	end int64
}

func (d *Downloader) runTask(tasks <-chan DownloadPartTask, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		if d.error != nil {
			return
		}

		partETag, err := d.downloadPart(task)
		if err != nil {
			d.setError(err)
			return
		}

		d.updatePart(partETag)
	}
}

func (d *Downloader) downloadPart(task DownloadPartTask) (CompletedPart, error) {
	request := d.downloadFileRequest
	dcp := d.downloadCheckpoint
	tempFilePath := dcp.DownloadFilePath + TempFileSuffix
	var completedPart CompletedPart
	resp, err := d.client.GetObjectWithContext(d.context, &GetObjectInput{
		Bucket:                     aws.String(dcp.BucketName),
		Key:                        aws.String(dcp.ObjectKey),
		Range:                      aws.String(fmt.Sprintf("bytes=%d-%d", task.start, task.end)),
		ResponseContentType:        request.ResponseContentType,
		ResponseContentLanguage:    request.ResponseContentLanguage,
		ResponseExpires:            request.ResponseExpires,
		ResponseCacheControl:       request.ResponseCacheControl,
		ResponseContentDisposition: request.ResponseContentDisposition,
		ResponseContentEncoding:    request.ResponseContentEncoding,
		IfModifiedSince:            request.IfModifiedSince,
		IfUnmodifiedSince:          request.IfUnmodifiedSince,
		IfMatch:                    request.IfMatch,
		IfNoneMatch:                request.IfNoneMatch,
		AcceptEncoding:             request.AcceptEncoding,
		SSECustomerAlgorithm:       request.SSECustomerAlgorithm,
		SSECustomerKey:             request.SSECustomerKey,
		SSECustomerKeyMD5:          request.SSECustomerKeyMD5,
	})

	if err != nil {
		return completedPart, err
	}
	defer resp.Body.Close()

	var crc64 hash.Hash64
	crc64 = crc.NewCRC(crc.CrcTable(), 0)
	resp.Body = aws.TeeReader(resp.Body, crc64, task.actualPartSize, nil)

	fd, err := os.OpenFile(tempFilePath, os.O_WRONLY|os.O_CREATE, FilePermMode)
	if err != nil {
		return completedPart, err
	}
	defer fd.Close()

	_, err = fd.Seek((task.partNumber-1)*dcp.PartSize, io.SeekStart)
	if err != nil {
		return completedPart, err
	}

	_, err = io.Copy(fd, resp.Body)
	if err != nil {
		return completedPart, err
	}

	completedPart.PartNumber = aws.Long(task.partNumber)
	completedPart.ChecksumCRC64ECMA = aws.String(strconv.FormatUint(crc64.Sum64(), 10))
	d.publishProgress(task.actualPartSize)

	return completedPart, nil
}

func (d *Downloader) updatePart(partETag CompletedPart) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.downloadCheckpoint.PartETagList = append(d.downloadCheckpoint.PartETagList, &partETag)
	d.downloadCheckpoint.dump()
}

func (d *Downloader) setError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.error == nil {
		d.error = err
	}
}

func (d *Downloader) publishProgress(actualPartSize int64) {
	if d.downloadFileRequest.ProgressFn != nil {
		atomic.AddInt64(&d.CompletedSize, actualPartSize)
		d.downloadFileRequest.ProgressFn(actualPartSize, d.CompletedSize, d.downloadFileSize)
	}
}

func (d *Downloader) getCrc64Ecma(parts []*CompletedPart) uint64 {
	if parts == nil || len(parts) == 0 {
		return 0
	}

	sort.Sort(CompletedParts(d.downloadCheckpoint.PartETagList))

	crcTemp, _ := strconv.ParseUint(*parts[0].ChecksumCRC64ECMA, 10, 64)
	for i := 1; i < len(parts); i++ {
		crc2, _ := strconv.ParseUint(*parts[i].ChecksumCRC64ECMA, 10, 64)
		partSize := d.getActualPartSize(d.downloadFileSize, aws.ToLong(d.downloadFileRequest.PartSize), *parts[i].PartNumber)
		crcTemp = crc.CRC64Combine(crcTemp, crc2, (uint64)(partSize))
	}

	return crcTemp
}

func (d *Downloader) complete() error {
	fileName := aws.ToString(d.downloadFileRequest.DownloadFile)
	tempFileName := fileName + TempFileSuffix
	err := os.Rename(tempFileName, fileName)
	if err != nil {
		return err
	}
	d.downloadCheckpoint.remove()
	return nil
}

func (d *Downloader) headObject() (map[string]*string, error) {
	request := d.downloadFileRequest
	resp, err := d.client.HeadObjectWithContext(d.context, &HeadObjectInput{
		Bucket:               request.Bucket,
		Key:                  request.Key,
		IfModifiedSince:      request.IfModifiedSince,
		IfUnmodifiedSince:    request.IfUnmodifiedSince,
		IfMatch:              request.IfMatch,
		IfNoneMatch:          request.IfNoneMatch,
		SSECustomerAlgorithm: request.SSECustomerAlgorithm,
		SSECustomerKey:       request.SSECustomerKey,
		SSECustomerKeyMD5:    request.SSECustomerKeyMD5,
	})
	if err != nil {
		return nil, err
	}
	return resp.Metadata, err
}

func (d *Downloader) createDownloadDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if !DirExists(dir) {
		err := os.MkdirAll(dir, DirPermMode)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Downloader) getObjectRange() []int64 {
	objectRange := d.downloadFileRequest.Range
	objectSize := d.downloadCheckpoint.ObjectSize
	if objectRange == nil {
		return []int64{0, objectSize - 1}
	}

	if !d.isValidRange(objectRange, objectSize) {
		d.client.Config.LogWarn("Invalid range value: %v, ignore it and request for entire object", objectRange)
		return []int64{0, objectSize - 1}
	}
	objectStart := objectRange[0]
	objectEnd := objectRange[1]

	if objectStart < 0 {
		return []int64{objectSize - objectEnd, objectSize - 1}
	}

	if objectEnd < 0 {
		return []int64{objectStart, objectSize - 1}
	}

	return []int64{objectStart, Min(objectEnd, objectSize-1)}
}

func (d *Downloader) isValidRange(objectRange []int64, objectSize int64) bool {
	if len(objectRange) != 2 {
		return false
	}

	objectStart := objectRange[0]
	objectEnd := objectRange[1]

	if objectStart < 0 && objectEnd < 0 || objectEnd >= 0 && objectStart > objectEnd {
		return false
	}

	return objectStart < objectSize
}

func (d *Downloader) normalizeDownloadPath() error {
	downloadPath := aws.ToString(d.downloadFileRequest.DownloadFile)
	if downloadPath == "" {
		downloadPath = aws.ToString(d.downloadFileRequest.Key)
	}
	// 规范化路径
	normalizedPath := filepath.Clean(downloadPath)
	// 获取绝对路径
	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		return err
	}
	d.downloadFileRequest.DownloadFile = aws.String(absPath)

	return nil
}
