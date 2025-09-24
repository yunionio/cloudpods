package s3

import (
	"context"
	"errors"
	"fmt"
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/crc"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type UploadFileInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Object key of the object.
	Key *string `location:"uri" locationName:"Key" type:"string" required:"true"`

	// The path of the file to be uploaded.
	UploadFile *string `type:"string" required:"true"`

	// The size of the file to be uploaded.
	FileSize *int64 `type:"integer"`

	// The file part fetcher.
	FilePartFetcher *FilePartFetcher `type:"structure"`

	// The object metadata.
	ObjectMeta map[string]*string `type:"structure"`

	// The size of each part.
	PartSize *int64 `type:"integer"`

	// The number of tasks to upload the file.
	TaskNum *int64 `type:"integer"`

	// Whether to enable checkpoint.
	EnableCheckpoint *bool `type:"boolean"`

	// The directory to store the checkpoint file.
	CheckpointDir *string `type:"string"`

	// The checkpoint file path.
	CheckpointFile *string `type:"string"`

	// The canned ACL to apply to the object.
	ACL *string `location:"header" locationName:"x-amz-acl" type:"string"`

	// Specifies caching behavior along the request/reply chain.
	CacheControl *string `location:"header" locationName:"Cache-Control" type:"string"`

	// Specifies presentational information for the object.
	ContentDisposition *string `location:"header" locationName:"Content-Disposition" type:"string"`

	// Specifies what content encodings have been applied to the object and thus
	// what decoding mechanisms must be applied to obtain the media-type referenced
	// by the Content-Type header field.
	ContentEncoding *string `location:"header" locationName:"Content-Encoding" type:"string"`

	// A standard MIME type describing the format of the object data.
	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// The date and time at which the object is no longer cacheable.
	Expires *time.Time `location:"header" locationName:"Expires" type:"timestamp" timestampFormat:"rfc822"`

	// A map of metadata to store with the object in S3.
	Metadata map[string]*string `location:"headers" locationName:"x-amz-meta-" type:"map"`

	// The type of storage to use for the object. Defaults to 'STANDARD'.
	StorageClass *string `location:"header" locationName:"x-amz-storage-class" type:"string"`

	// Specifies the object tag of the object. Multiple tags can be set at the same time, such as: TagA=A&TagB=B.
	// Note: Key and Value need to be URL-encoded first. If an item does not have "=", the Value is considered to be an empty string.
	Tagging *string `location:"header" locationName:"x-amz-tagging" type:"string"`

	// Specifies whether the object is forbidden to overwrite.
	ForbidOverwrite *bool `location:"header" locationName:"x-amz-forbid-overwrite" type:"boolean"`

	// Allows grantee to read the object data and its metadata.
	GrantRead *string `location:"header" locationName:"x-amz-grant-read" type:"string"`

	// Gives the grantee READ, READ_ACP, and WRITE_ACP permissions on the object.
	GrantFullControl *string `location:"header" locationName:"x-amz-grant-full-control" type:"string"`

	// The Server-side encryption algorithm used when storing this object in KS3, eg: AES256.
	ServerSideEncryption *string `location:"header" locationName:"x-amz-server-side-encryption" type:"string"`

	// Specifies the algorithm to use to when encrypting the object, eg: AES256.
	SSECustomerAlgorithm *string `location:"header" locationName:"x-amz-server-side-encryption-customer-algorithm" type:"string"`

	// Specifies the customer-provided encryption key for KS3 to use in encrypting data.
	SSECustomerKey *string `location:"header" locationName:"x-amz-server-side-encryption-customer-key" type:"string"`

	// Specifies the 128-bit MD5 digest of the encryption key according to RFC 1321.
	SSECustomerKeyMD5 *string `location:"header" locationName:"x-amz-server-side-encryption-customer-key-MD5" type:"string"`

	// Progress callback function
	ProgressFn aws.ProgressFunc `location:"function"`
}

type UploadFileOutput struct {
	Bucket *string

	Key *string

	ETag *string

	ChecksumCRC64ECMA *string
}

type FilePartFetcher interface {
	Fetch(objectRange []int64) (io.ReadSeeker, error)
}

func (c *S3) UploadFile(request *UploadFileInput) (*UploadFileOutput, error) {
	return c.UploadFileWithContext(context.Background(), request)
}

func (c *S3) UploadFileWithContext(ctx context.Context, request *UploadFileInput) (*UploadFileOutput, error) {
	return newUploader(c, ctx, request).uploadFile()
}

type Uploader struct {
	client *S3

	context context.Context

	uploadFileRequest *UploadFileInput

	uploadCheckpoint *UploadCheckpoint

	CompletedSize int64

	mu sync.Mutex

	error error
}

func newUploader(s3 *S3, ctx context.Context, request *UploadFileInput) *Uploader {
	return &Uploader{
		client:            s3,
		context:           ctx,
		uploadFileRequest: request,
	}
}

func (u *Uploader) uploadFile() (*UploadFileOutput, error) {
	err := u.validate()
	if err != nil {
		return nil, err
	}

	if aws.ToString(u.uploadFileRequest.UploadFile) != "" && aws.ToLong(u.uploadFileRequest.FileSize) <= aws.ToLong(u.uploadFileRequest.PartSize) {
		return u.putObject()
	}

	return u.multipartUpload()
}

func (u *Uploader) validate() error {
	request := u.uploadFileRequest
	if request == nil {
		return errors.New("upload file request is required")
	}

	if aws.ToString(request.Bucket) == "" {
		return errors.New("bucket is required")
	}

	if aws.ToString(request.Key) == "" {
		return errors.New("key is required")
	}

	err := u.normalizeUploadPath()
	if err != nil {
		return err
	}

	filePath := aws.ToString(request.UploadFile)
	if filePath == "" && request.FilePartFetcher == nil {
		return errors.New("upload file or file part fetcher is required")
	}

	if filePath != "" {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			return errors.New("upload file not a file")
		}
		request.FileSize = aws.Long(fileInfo.Size())
	} else {
		if request.ObjectMeta != nil {
			fileSize, _ := strconv.ParseInt(aws.ToString(request.ObjectMeta[HTTPHeaderContentLength]), 10, 64)
			request.FileSize = aws.Long(fileSize)
		}
	}

	if request.FilePartFetcher != nil && request.FileSize == nil {
		return errors.New("file size is required")
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

func (u *Uploader) putObject() (*UploadFileOutput, error) {
	request := u.uploadFileRequest
	fd, err := os.Open(aws.ToString(request.UploadFile))
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	resp, err := u.client.PutObjectWithContext(u.context, &PutObjectInput{
		Bucket:               request.Bucket,
		Key:                  request.Key,
		Body:                 fd,
		ACL:                  request.ACL,
		CacheControl:         request.CacheControl,
		ContentDisposition:   request.ContentDisposition,
		ContentEncoding:      request.ContentEncoding,
		ContentType:          request.ContentType,
		Expires:              request.Expires,
		Metadata:             request.Metadata,
		StorageClass:         request.StorageClass,
		Tagging:              request.Tagging,
		ForbidOverwrite:      request.ForbidOverwrite,
		GrantRead:            request.GrantRead,
		GrantFullControl:     request.GrantFullControl,
		ServerSideEncryption: request.ServerSideEncryption,
		SSECustomerAlgorithm: request.SSECustomerAlgorithm,
		SSECustomerKey:       request.SSECustomerKey,
		SSECustomerKeyMD5:    request.SSECustomerKeyMD5,
		ProgressFn:           request.ProgressFn,
	})
	if err != nil {
		return nil, err
	}

	return &UploadFileOutput{
		Bucket:            request.Bucket,
		Key:               request.Key,
		ETag:              resp.ETag,
		ChecksumCRC64ECMA: resp.Metadata[HTTPHeaderAmzChecksumCrc64ecma],
	}, nil
}

func (u *Uploader) multipartUpload() (*UploadFileOutput, error) {
	ucp, err := newUploadCheckpoint(u)
	if err != nil {
		return nil, err
	}
	u.uploadCheckpoint = ucp

	if aws.ToBoolean(u.uploadFileRequest.EnableCheckpoint) {
		cpFilePath := aws.ToString(u.uploadFileRequest.CheckpointFile)
		if cpFilePath == "" {
			cpFilePath, err = generateUploadCpFilePath(u.uploadFileRequest)
			if err != nil {
				return nil, err
			}
		}
		ucp.CpFilePath = cpFilePath

		err = ucp.load()
		if err != nil {
			return nil, err
		}

		if ucp.UploadId != "" && !u.isUploadIdValid() {
			ucp.UploadId = ""
			ucp.PartETagList = make([]*CompletedPart, 0)
			ucp.remove()
		}
	}

	if ucp.UploadId == "" {
		ucp.UploadId, err = u.initUploadId()
		if err != nil {
			return nil, err
		}
		ucp.dump()
	}

	fileSize := ucp.UploadFileSize
	partSize := ucp.PartSize
	totalPartNum := (fileSize-1)/partSize + 1
	tasks := make(chan UploadPartTask, totalPartNum)

	var i int64
	for i = 0; i < totalPartNum; i++ {
		partNum := i + 1
		offset := i * partSize
		actualPartSize := u.getActualPartSize(fileSize, partSize, partNum)
		partETag := u.getPartETag(partNum)
		if partETag != nil {
			u.publishProgress(actualPartSize)
		} else {
			uploadPartTask := UploadPartTask{
				partNumber:     partNum,
				offset:         offset,
				actualPartSize: actualPartSize,
			}
			tasks <- uploadPartTask
		}
	}
	close(tasks)

	var wg sync.WaitGroup
	for i = 0; i < aws.ToLong(u.uploadFileRequest.TaskNum); i++ {
		wg.Add(1)
		go u.runTask(tasks, &wg)
	}
	wg.Wait()

	if u.error != nil {
		return nil, u.error
	}

	completedMultipartUpload := u.getMultipartUploadParts()
	resp, err := u.completeMultipartUpload(completedMultipartUpload)
	if err != nil {
		return nil, err
	}

	if u.client.Config.CrcCheckEnabled {
		clientCrc64 := u.getCrc64Ecma(completedMultipartUpload.Parts)
		serverCrc64, _ := strconv.ParseUint(aws.ToString(resp.ChecksumCRC64ECMA), 10, 64)
		u.client.Config.LogDebug("check file crc64, client crc64:%d, server crc64:%d", clientCrc64, serverCrc64)
		if serverCrc64 != 0 && clientCrc64 != serverCrc64 {
			return nil, errors.New(fmt.Sprintf("crc64 check failed, client crc64:%d, server crc64:%d", clientCrc64, serverCrc64))
		}
	}

	return u.getUploadFileOutput(resp), nil
}

func (u *Uploader) getUploadFileOutput(resp *CompleteMultipartUploadOutput) *UploadFileOutput {
	return &UploadFileOutput{
		Bucket:            resp.Bucket,
		Key:               resp.Key,
		ETag:              resp.ETag,
		ChecksumCRC64ECMA: resp.ChecksumCRC64ECMA,
	}
}

func (u *Uploader) getPartSize(fileSize int64, originPartSize int64) int64 {
	partSize := originPartSize
	totalPartNum := (fileSize-1)/partSize + 1
	for totalPartNum > MaxPartNum {
		partSize += originPartSize
		totalPartNum = (fileSize-1)/partSize + 1
	}
	return partSize
}

func (u *Uploader) getActualPartSize(fileSize int64, partSize int64, partNum int64) int64 {
	offset := (partNum - 1) * partSize
	actualPartSize := partSize
	if offset+partSize >= fileSize {
		actualPartSize = fileSize - offset
	}
	return actualPartSize
}

func (u *Uploader) getPartETag(partNumber int64) *CompletedPart {
	for _, partETag := range u.uploadCheckpoint.PartETagList {
		if *partETag.PartNumber == partNumber {
			return partETag
		}
	}
	return nil
}

type UploadPartTask struct {
	partNumber int64

	offset int64

	actualPartSize int64
}

func (u *Uploader) runTask(tasks <-chan UploadPartTask, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		if u.error != nil {
			return
		}

		partETag, err := u.uploadPart(task)
		if err != nil {
			u.setError(err)
			return
		}

		u.updatePart(partETag)
	}
}

func (u *Uploader) uploadPart(task UploadPartTask) (CompletedPart, error) {
	request := u.uploadFileRequest
	ucp := u.uploadCheckpoint
	offset := task.offset
	actualPartSize := task.actualPartSize
	var partETag CompletedPart
	var reader io.ReadSeeker
	if ucp.UploadFilePath != "" {
		fd, err := os.Open(ucp.UploadFilePath)
		if err != nil {
			return partETag, err
		}
		defer fd.Close()

		reader = io.NewSectionReader(fd, offset, actualPartSize)
	} else {
		var err error
		reader, err = (*u.uploadFileRequest.FilePartFetcher).Fetch([]int64{offset, offset + actualPartSize - 1})
		if err != nil {
			return partETag, err
		}
	}

	resp, err := u.client.UploadPartWithContext(u.context, &UploadPartInput{
		Bucket:               aws.String(ucp.BucketName),
		Key:                  aws.String(ucp.ObjectKey),
		UploadID:             aws.String(ucp.UploadId),
		PartNumber:           aws.Long(task.partNumber),
		Body:                 reader,
		ContentLength:        aws.Long(actualPartSize),
		SSECustomerAlgorithm: request.SSECustomerAlgorithm,
		SSECustomerKey:       request.SSECustomerKey,
		SSECustomerKeyMD5:    request.SSECustomerKeyMD5,
	})

	if err != nil {
		return partETag, err
	}

	partETag.PartNumber = aws.Long(task.partNumber)
	partETag.ETag = resp.ETag
	partETag.ChecksumCRC64ECMA = resp.ChecksumCRC64ECMA
	u.publishProgress(actualPartSize)

	return partETag, nil
}

func (u *Uploader) updatePart(partETag CompletedPart) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.uploadCheckpoint.PartETagList = append(u.uploadCheckpoint.PartETagList, &partETag)
	u.uploadCheckpoint.dump()
}

func (u *Uploader) setError(err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.error == nil {
		u.error = err
	}
}

type CompletedParts []*CompletedPart

func (cp CompletedParts) Len() int {
	return len(cp)
}

func (cp CompletedParts) Less(i, j int) bool {
	return *cp[i].PartNumber < *cp[j].PartNumber
}

func (cp CompletedParts) Swap(i, j int) {
	cp[i], cp[j] = cp[j], cp[i]
}

func (u *Uploader) getMultipartUploadParts() *CompletedMultipartUpload {
	partETags := u.uploadCheckpoint.PartETagList
	// 按照PartNumber排序
	sort.Sort(CompletedParts(partETags))
	return &CompletedMultipartUpload{
		Parts: partETags,
	}
}

func (u *Uploader) completeMultipartUpload(completedMultipartUpload *CompletedMultipartUpload) (*CompleteMultipartUploadOutput, error) {
	resp, err := u.client.CompleteMultipartUploadWithContext(u.context, &CompleteMultipartUploadInput{
		Bucket:          u.uploadFileRequest.Bucket,
		Key:             u.uploadFileRequest.Key,
		UploadID:        aws.String(u.uploadCheckpoint.UploadId),
		MultipartUpload: completedMultipartUpload,
		ForbidOverwrite: u.uploadFileRequest.ForbidOverwrite,
	})
	if err != nil {
		return nil, err
	}
	u.uploadCheckpoint.remove()
	return resp, err
}

func (u *Uploader) publishProgress(actualPartSize int64) {
	if u.uploadFileRequest.ProgressFn != nil {
		atomic.AddInt64(&u.CompletedSize, actualPartSize)
		u.uploadFileRequest.ProgressFn(actualPartSize, u.CompletedSize, aws.ToLong(u.uploadFileRequest.FileSize))
	}
}

func (u *Uploader) getCrc64Ecma(parts []*CompletedPart) uint64 {
	if parts == nil || len(parts) == 0 {
		return 0
	}

	fileSize := u.uploadCheckpoint.UploadFileSize
	partSize := u.uploadCheckpoint.PartSize

	crcTemp, _ := strconv.ParseUint(*parts[0].ChecksumCRC64ECMA, 10, 64)
	for i := 1; i < len(parts); i++ {
		crc2, _ := strconv.ParseUint(*parts[i].ChecksumCRC64ECMA, 10, 64)
		actualPartSize := u.getActualPartSize(fileSize, partSize, *parts[i].PartNumber)
		crcTemp = crc.CRC64Combine(crcTemp, crc2, (uint64)(actualPartSize))
	}

	return crcTemp
}

func (u *Uploader) initUploadId() (string, error) {
	request := u.uploadFileRequest
	resp, err := u.client.CreateMultipartUploadWithContext(u.context, &CreateMultipartUploadInput{
		Bucket:               request.Bucket,
		Key:                  request.Key,
		ACL:                  request.ACL,
		CacheControl:         request.CacheControl,
		ContentDisposition:   request.ContentDisposition,
		ContentEncoding:      request.ContentEncoding,
		ContentType:          request.ContentType,
		Expires:              request.Expires,
		Metadata:             request.Metadata,
		StorageClass:         request.StorageClass,
		Tagging:              request.Tagging,
		ForbidOverwrite:      request.ForbidOverwrite,
		GrantRead:            request.GrantRead,
		GrantFullControl:     request.GrantFullControl,
		ServerSideEncryption: request.ServerSideEncryption,
		SSECustomerAlgorithm: request.SSECustomerAlgorithm,
		SSECustomerKey:       request.SSECustomerKey,
		SSECustomerKeyMD5:    request.SSECustomerKeyMD5,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.UploadID), nil
}

func (u *Uploader) isUploadIdValid() bool {
	_, err := u.client.ListPartsWithContext(u.context, &ListPartsInput{
		Bucket:   u.uploadFileRequest.Bucket,
		Key:      u.uploadFileRequest.Key,
		UploadID: aws.String(u.uploadCheckpoint.UploadId),
	})
	if err != nil && strings.Contains(err.Error(), "NoSuchUpload") {
		return false
	}
	return true
}

func (u *Uploader) normalizeUploadPath() error {
	uploadPath := aws.ToString(u.uploadFileRequest.UploadFile)
	if uploadPath == "" {
		return nil
	}
	// 规范化路径
	normalizedPath := filepath.Clean(uploadPath)
	// 获取绝对路径
	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		return err
	}
	u.uploadFileRequest.UploadFile = aws.String(absPath)

	return nil
}
