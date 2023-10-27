package tos

import (
	"context"
	"encoding/json"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

const DefaultProgressCallbackSize = 512 * 1024

type Metadata interface {
	AllKeys() []string
	Get(key string) (string, bool)
	Range(f func(key, value string) bool)
}

type CustomMeta struct {
	m map[string]string
}

func (c *CustomMeta) AllKeys() []string {
	keys := make([]string, 0, len(c.m))
	for k := range c.m {
		keys = append(keys, k)
	}
	return keys
}

func (c *CustomMeta) Get(key string) (val string, ok bool) {
	val, ok = c.m[strings.ToLower(key)]
	return
}

func (c *CustomMeta) Range(f func(key, val string) bool) {
	for k, v := range c.m {
		if !f(k, v) {
			break
		}
	}
}

type multipartUpload struct {
	Bucket   string `json:"Bucket,omitempty"`
	Key      string `json:"Key,omitempty"`
	UploadID string `json:"UploadId,omitempty"`
}

type uploadedPart struct {
	PartNumber int    `json:"PartNumber"`
	ETag       string `json:"ETag"`
}

type uploadedParts []uploadedPart

func (p uploadedParts) Less(i, j int) bool { return p[i].PartNumber < p[j].PartNumber }
func (p uploadedParts) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p uploadedParts) Len() int           { return len(p) }

// only for marshal
type partsToComplete struct {
	Parts uploadedParts `json:"Parts"`
}

// only for Marshal
type deleteMultiObjectsInput struct {
	Objects []ObjectTobeDeleted `json:"Objects,omitempty"`
	Quiet   bool                `json:"Quiet,omitempty"`
}

// only for Marshal
type accessControlList struct {
	Owner                Owner     `json:"Owner,omitempty"`
	Grants               []GrantV2 `json:"Grants,omitempty"`
	BucketOwnerEntrusted bool      `json:"BucketOwnerEntrusted,omitempty"`
}

type canceler struct {
	called       int32
	cancelHandle chan struct{}
	// cleaner will clean all files need to be deleted
	cleaner func()
	// aborter will  abort multi upload task
	aborter func() error
}

func (c *canceler) Cancel(isAbort bool) {
	if c.cancelHandle == nil {
		return
	}
	if atomic.CompareAndSwapInt32(&c.called, 0, 1) {
		if isAbort {
			if c.cleaner != nil {
				c.cleaner()
			}
			if c.aborter != nil {
				c.aborter()
			}
		}
		close(c.cancelHandle)
	}
}

// do nothing
func (c *canceler) internal() {}

type copyPartInfo struct {
	PartNumber           int64  `json:"part_number"`
	CopySourceRange      string `json:"copy_source_range"`
	CopySourceRangeStart int64  `json:"copy_source_range_start"`
	CopySourceRangeEnd   int64  `json:"copy_source_range_end"`
	Etag                 string `json:"etag"`
	IsCompleted          bool   `json:"is_completed"`
	IsZeroSize           bool   `json:"is_zero_size"`
}

type copyObjectCheckpoint struct {
	Bucket                      string         `json:"bucket"`
	Key                         string         `json:"key"`
	SrcBucket                   string         `json:"src_bucket"`
	SrcVersionID                string         `json:"src_version_id"`
	PartSize                    int64          `json:"part_size"`
	UploadID                    string         `json:"upload_id"`
	CopySourceIfMatch           string         `json:"copy_source_if_match"`
	CopySourceIfModifiedSince   time.Time      `json:"copy_source_if_modified_since"`
	CopySourceIfNoneMatch       string         `json:"copy_source_if_none_match"`
	CopySourceIfUnmodifiedSince time.Time      `json:"copy_source_if_unmodified_since"`
	CopySourceSSECAlgorithm     string         `json:"copy_source_ssec_algorithm"`
	CopySourceSSECKeyMD5        string         `json:"copy_source_ssec_key_md5"`
	SSECAlgorithm               string         `json:"ssec_algorithm"`
	SSECKeyMD5                  string         `json:"ssec_key_md5"`
	EncodingType                string         `json:"encoding_type"`
	CopySourceObjectInfo        objectInfo     `json:"copy_source_object_info"`
	PartsInfo                   []copyPartInfo `json:"parts_info"`
	CheckpointPath              string         `json:"checkpoint_path"`
}

func (c *copyObjectCheckpoint) Valid(input *ResumableCopyObjectInput, headOutput *HeadObjectV2Output) bool {
	// 源对象发生改变
	if c.CopySourceObjectInfo.ObjectSize != headOutput.ContentLength || c.CopySourceObjectInfo.Etag != headOutput.ETag ||
		c.CopySourceObjectInfo.LastModified != headOutput.LastModified || c.CopySourceObjectInfo.HashCrc64ecma != headOutput.HashCrc64ecma {
		return false
	}
	// 复制基本信息发生改变
	if c.Bucket != input.Bucket || input.Key != c.Key ||
		input.SrcBucket != c.SrcBucket || input.SrcVersionID != c.SrcVersionID ||
		input.PartSize != c.PartSize || input.EncodingType != c.EncodingType {
		return false
	}

	// 复制条件发生改变
	if c.CopySourceIfMatch != input.CopySourceIfMatch || c.CopySourceIfModifiedSince != input.CopySourceIfModifiedSince ||
		c.CopySourceIfUnmodifiedSince != input.CopySourceIfUnmodifiedSince || c.CopySourceIfNoneMatch != input.CopySourceIfNoneMatch {
		return false
	}
	// 加密发生改变
	if c.SSECAlgorithm != input.SSECAlgorithm || c.SSECKeyMD5 != input.SSECKeyMD5 ||
		c.CopySourceSSECAlgorithm != input.CopySourceSSECAlgorithm || c.CopySourceSSECKeyMD5 != input.CopySourceSSECKeyMD5 {
		return false
	}
	return true
}
func (c *copyObjectCheckpoint) WriteToFile() error {

	buffer, err := json.Marshal(c)
	if err != nil {
		return InvalidMarshal
	}
	err = ioutil.WriteFile(c.CheckpointPath, buffer, 0600)
	if err != nil {
		return newTosClientError(err.Error(), err)
	}
	return nil
}

func (c *copyObjectCheckpoint) UpdatePartsInfo(result interface{}) {
	part := result.(copyPartInfo)
	c.PartsInfo[part.PartNumber-1] = part
}

func (c *copyObjectCheckpoint) GetCheckPointFilePath() string {
	return c.CheckpointPath
}

func (c *copyObjectCheckpoint) GetParts() []UploadedPartV2 {
	parts := make([]UploadedPartV2, 0, len(c.PartsInfo))
	for _, p := range c.PartsInfo {
		parts = append(parts, UploadedPartV2{
			PartNumber: int(p.PartNumber),
			ETag:       p.Etag,
		})
	}
	return parts
}

type objectInfo struct {
	Etag          string    `json:"Etag,omitempty"`
	HashCrc64ecma uint64    `json:"HashCrc64Ecma,omitempty"`
	LastModified  time.Time `json:"LastModified,omitempty"`
	ObjectSize    int64     `json:"ObjectSize,omitempty"`
}

type downloadFileInfo struct {
	FilePath     string `json:"FilePath,omitempty"`
	TempFilePath string `json:"TempFilePath,omitempty"`
}

// downloadPartInfo is for checkpoint
type downloadPartInfo struct {
	PartNumber    int    `json:"PartNumber,omitempty"`
	RangeStart    int64  `json:"RangeStart"` // not omit empty
	RangeEnd      int64  `json:"RangeEnd,omitempty"`
	HashCrc64ecma uint64 `json:"HashCrc64Ecma,omitempty"`
	IsCompleted   bool   `json:"IsCompleted"` // not omit empty
}

type downloadCheckpoint struct {
	checkpointPath string // this filed should not be marshaled
	Bucket         string `json:"Bucket,omitempty"`
	Key            string `json:"Key,omitempty"`
	VersionID      string `json:"VersionID,omitempty"`
	PartSize       int64  `json:"PartSize,omitempty"`

	IfMatch           string    `json:"IfMatch,omitempty"`
	IfModifiedSince   time.Time `json:"IfModifiedSince,omitempty"`
	IfNoneMatch       string    `json:"IfNoneMatch,omitempty"`
	IfUnmodifiedSince time.Time `json:"IfUnmodifiedSince,omitempty"`

	SSECAlgorithm string             `json:"SSECAlgorithm,omitempty"`
	SSECKeyMD5    string             `json:"SSECKeyMD5,omitempty"`
	ObjectInfo    objectInfo         `json:"ObjectInfo,omitempty"`
	FileInfo      downloadFileInfo   `json:"FileInfo,omitempty"`
	PartsInfo     []downloadPartInfo `json:"PartsInfo,omitempty"`
}

func (c *downloadCheckpoint) UpdatePartsInfo(result interface{}) {
	part := result.(downloadPartInfo)
	c.PartsInfo[part.PartNumber-1] = part

}

func (c *downloadCheckpoint) GetCheckPointFilePath() string {
	return c.checkpointPath
}

func (c *downloadCheckpoint) WriteToFile() error {
	buffer, err := json.Marshal(c)
	if err != nil {
		return InvalidMarshal
	}
	err = ioutil.WriteFile(c.checkpointPath, buffer, 0600)
	if err != nil {
		return newTosClientError(err.Error(), err)
	}
	return nil
}

func (c *downloadCheckpoint) Valid(input *DownloadFileInput, head *HeadObjectV2Output) bool {
	if c.Bucket != input.Bucket || c.Key != input.Key || c.VersionID != input.VersionID || c.PartSize != input.PartSize ||
		c.IfMatch != input.IfMatch || c.IfModifiedSince != input.IfModifiedSince || c.IfNoneMatch != input.IfNoneMatch ||
		c.IfUnmodifiedSince != input.IfUnmodifiedSince ||
		c.SSECAlgorithm != input.SSECAlgorithm || c.SSECKeyMD5 != input.SSECKeyMD5 {
		return false
	}

	if c.ObjectInfo.Etag != head.ETag || c.ObjectInfo.HashCrc64ecma != head.HashCrc64ecma ||
		c.ObjectInfo.LastModified != head.LastModified || c.ObjectInfo.ObjectSize != head.ContentLength {
		return false
	}
	if c.FileInfo.FilePath != input.filePath {
		return false
	}
	return true
}

type fileInfo struct {
	LastModified int64 `json:"LastModified,omitempty"`
	Size         int64 `json:"Size"`
}

// uploadPartInfo is for checkpoint
type uploadPartInfo struct {
	uploadID      *string // should not be marshaled
	PartNumber    int     `json:"PartNumber"`
	PartSize      int64   `json:"PartSize"`
	Offset        uint64  `json:"Offset"`
	ETag          string  `json:"ETag,omitempty"`
	HashCrc64ecma uint64  `json:"HashCrc64Ecma,omitempty"`
	IsCompleted   bool    `json:"IsCompleted"`
}

type uploadCheckpoint struct {
	checkpointPath string           // this filed should not be marshaled
	Bucket         string           `json:"Bucket,omitempty"`
	Key            string           `json:"Key,omitempty"`
	UploadID       string           `json:"UploadID,omitempty"`
	PartSize       int64            `json:"PartSize"`
	SSECAlgorithm  string           `json:"SSECAlgorithm,omitempty"`
	SSECKeyMD5     string           `json:"SSECKeyMD5,omitempty"`
	EncodingType   string           `json:"EncodingType,omitempty"`
	FilePath       string           `json:"FilePath,omitempty"`
	FileInfo       fileInfo         `json:"FileInfo"`
	PartsInfo      []uploadPartInfo `json:"PartsInfo,omitempty"`
}

func (u *uploadCheckpoint) UpdatePartsInfo(result interface{}) {
	part := result.(uploadPartInfo)
	u.PartsInfo[part.PartNumber-1] = part

}

func (u *uploadCheckpoint) GetCheckPointFilePath() string {
	return u.FilePath
}

func (u *uploadCheckpoint) Valid(uploadFileStat os.FileInfo, bucketName, key, uploadFile string) bool {
	if u.UploadID == "" || u.Bucket != bucketName || u.Key != key || u.FilePath != uploadFile {
		return false
	}
	if u.FileInfo.Size != uploadFileStat.Size() || u.FileInfo.LastModified != uploadFileStat.ModTime().Unix() {
		return false
	}
	return true
}

func (u *uploadCheckpoint) GetParts() []UploadedPartV2 {
	parts := make([]UploadedPartV2, 0, len(u.PartsInfo))
	for _, p := range u.PartsInfo {
		parts = append(parts, UploadedPartV2{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}
	return parts
}

func (u *uploadCheckpoint) WriteToFile() error {
	result, err := json.Marshal(u)
	if err != nil {
		return InvalidMarshal
	}
	err = ioutil.WriteFile(u.checkpointPath, result, 0600)
	if err != nil {
		return newTosClientError(err.Error(), err)
	}
	return nil
}

type copyEvent struct {
	input    *ResumableCopyObjectInput
	uploadID string
}

func (c *copyEvent) postCopyEvent(event *CopyEvent) {
	if c.input.CopyEventListener != nil {
		c.input.CopyEventListener.EventChange(event)
	}
}
func (c *copyEvent) PostEvent(eventType int, result interface{}, taskErr error) {

	event := &CopyEvent{
		Bucket:         c.input.Bucket,
		Key:            c.input.Key,
		UploadID:       &c.uploadID,
		SrcBucket:      c.input.SrcBucket,
		SrcKey:         c.input.SrcKey,
		SrcVersionID:   c.input.SrcVersionID,
		CheckpointFile: &c.input.CheckpointFile,
	}
	switch eventType {
	case EventPartSucceed:
		part, ok := result.(copyPartInfo)
		if !ok {
			return
		}
		event.CopyPartInfo = &part
		event.Type = enum.CopyEventUploadPartCopySuccess
		c.postCopyEvent(event)
	case EventPartFailed:
		event.Type = enum.CopyEventUploadPartCopyFailed
		c.postCopyEvent(event)
	case EventPartAborted:
		event.Type = enum.CopyEventUploadPartCopyAborted
		event.Err = taskErr
		c.postCopyEvent(event)
	default:
	}
}

type downloadEvent struct {
	input *DownloadFileInput
}

func (d downloadEvent) PostEvent(eventType int, result interface{}, taskErr error) {
	switch eventType {
	case EventPartSucceed:
		part, ok := result.(downloadPartInfo)
		if !ok {
			return
		}
		d.postDownloadEvent(d.newDownloadPartSucceedEvent(part))
	case EventPartFailed:
		d.postDownloadEvent(d.newFailedEvent(taskErr, enum.DownloadEventDownloadPartFailed))
	case EventPartAborted:
		d.postDownloadEvent(d.newFailedEvent(taskErr, enum.DownloadEventDownloadPartAborted))
	default:
	}
}

type downloadTask struct {
	cli         *ClientV2
	ctx         context.Context
	input       *DownloadFileInput
	consumed    *int64
	subtotal    *int64
	total       int64
	partNumber  int
	rangeStart  int64
	rangeEnd    int64
	enableCRC64 bool
}

// Do the downloadTask, and return downloadPartInfo
func (t *downloadTask) do() (result interface{}, err error) {
	input := t.getBaseInput().(GetObjectV2Input)
	output, err := t.cli.GetObjectV2(t.ctx, &input)
	if err != nil {
		return nil, err
	}
	defer output.Content.Close()
	file, err := os.OpenFile(t.input.tempFile, os.O_RDWR, DefaultFilePerm)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	var wrapped = output.Content
	if t.input.DataTransferListener != nil {
		wrapped = &parallelReadCloserWithListener{
			listener: t.input.DataTransferListener,
			base:     wrapped,
			consumed: t.consumed,
			total:    t.total,
			subtotal: t.subtotal,
		}
	}
	if t.input.RateLimiter != nil {
		wrapped = &ReadCloserWithLimiter{
			limiter: t.input.RateLimiter,
			base:    wrapped,
		}
	}
	var checker hash.Hash64
	if t.enableCRC64 {
		checker = crc64.New(crc64.MakeTable(crc64.ECMA))
		wrapped = &readCloserWithCRC{
			checker: checker,
			base:    wrapped,
		}
	}

	_, err = file.Seek(t.rangeStart, io.SeekStart)
	if err != nil {
		return nil, err
	}
	written, err := io.Copy(file, wrapped)
	if err != nil {
		return nil, err
	}
	if written != (t.rangeEnd - t.rangeStart + 1) {
		return nil, fmt.Errorf("io copy want length %d but get %d. ", t.rangeEnd-t.rangeStart+1, written)
	}
	part := downloadPartInfo{
		PartNumber:  t.partNumber,
		RangeStart:  t.rangeStart,
		RangeEnd:    t.rangeEnd,
		IsCompleted: true,
	}
	if t.enableCRC64 {
		part.HashCrc64ecma = checker.Sum64()
	}
	return part, nil
}

func (t *downloadTask) getBaseInput() interface{} {
	return GetObjectV2Input{
		Bucket:            t.input.Bucket,
		Key:               t.input.Key,
		VersionID:         t.input.VersionID,
		IfMatch:           t.input.IfMatch,
		IfModifiedSince:   t.input.IfModifiedSince,
		IfNoneMatch:       t.input.IfNoneMatch,
		IfUnmodifiedSince: t.input.IfUnmodifiedSince,
		SSECAlgorithm:     t.input.SSECAlgorithm,
		SSECKey:           t.input.SSECKey,
		SSECKeyMD5:        t.input.SSECKeyMD5,
		RangeStart:        t.rangeStart,
		RangeEnd:          t.rangeEnd,
		TrafficLimit:      t.input.TrafficLimit,
		// we want to Sent parallel Listener on output, so explicitly set listener of GetObjectV2Input nil here.
		DataTransferListener: nil,
		RateLimiter:          nil,
	}
}

type uploadPostEvent struct {
	input      *UploadFileInput
	checkPoint *uploadCheckpoint
}

func (u *uploadPostEvent) PostEvent(eventType int, result interface{}, taskErr error) {
	switch eventType {
	case EventPartSucceed:
		partInfo, ok := result.(uploadPartInfo)
		if !ok {
			return
		}
		u.postUploadEvent(u.newUploadPartSucceedEvent(u.input, partInfo))
	case EventPartFailed:
		u.postUploadEvent(u.newUploadPartFailedEvent(u.input, u.checkPoint.UploadID, taskErr))
	case EventPartAborted:
		u.postUploadEvent(u.newUploadPartAbortedEvent(u.input, u.checkPoint.UploadID, taskErr))

	}
}

type uploadTask struct {
	cli        *ClientV2
	input      *UploadFileInput
	consumed   *int64
	subtotal   *int64
	mutex      *sync.Mutex
	ctx        context.Context
	total      int64
	UploadID   string
	ContentMD5 string
	PartNumber int
	Offset     uint64
	PartSize   int64
}

// Do the uploadTask, and return uploadPartInfo
func (t *uploadTask) do() (interface{}, error) {
	file, err := os.Open(t.input.FilePath)
	if err != nil {
		return nil, newTosClientError(err.Error(), err)
	}
	_, err = file.Seek(int64(t.Offset), io.SeekStart)
	if err != nil {
		return nil, newTosClientError(err.Error(), err)
	}
	var wrapped = ioutil.NopCloser(io.LimitReader(file, t.input.PartSize))
	if t.input.DataTransferListener != nil {
		wrapped = &parallelReadCloserWithListener{
			listener: t.input.DataTransferListener,
			base:     wrapped,
			total:    t.total,
			subtotal: t.subtotal,
			consumed: t.consumed,
		}
	}
	if t.input.RateLimiter != nil {
		wrapped = &ReadCloserWithLimiter{
			limiter: t.input.RateLimiter,
			base:    wrapped,
		}
	}
	input := t.getBaseInput().(UploadPartV2Input)
	input.Content = wrapped
	output, err := t.cli.UploadPartV2(t.ctx, &UploadPartV2Input{
		UploadPartBasicInput: input.UploadPartBasicInput,
		Content:              wrapped,
		ContentLength:        input.ContentLength,
	})
	if err != nil {
		return nil, err
	}
	return uploadPartInfo{
		uploadID:      &t.UploadID,
		PartNumber:    output.PartNumber,
		PartSize:      t.PartSize,
		Offset:        t.Offset,
		ETag:          output.ETag,
		HashCrc64ecma: output.HashCrc64ecma,
		IsCompleted:   true,
	}, nil
}

func (t *uploadTask) getBaseInput() interface{} {
	return UploadPartV2Input{
		UploadPartBasicInput: UploadPartBasicInput{
			Bucket:               t.input.Bucket,
			Key:                  t.input.Key,
			UploadID:             t.UploadID,
			PartNumber:           t.PartNumber,
			ContentMD5:           t.ContentMD5,
			SSECAlgorithm:        t.input.SSECAlgorithm,
			SSECKey:              t.input.SSECKey,
			SSECKeyMD5:           t.input.SSECKeyMD5,
			ServerSideEncryption: t.input.ServerSideEncryption,
			TrafficLimit:         t.input.TrafficLimit,
		},
		ContentLength: t.PartSize,
	}
}

type retryAction int

const (
	NoRetry retryAction = iota
	Retry
)

const (
	DefaultRetryBackoffBase = 100 * time.Millisecond
	DefaultRetryTime        = 3
)

type classifier interface {
	Classify(error) retryAction
}

func exponentialBackoff(n int, base time.Duration) []time.Duration {
	backoffs := make([]time.Duration, n)
	for i := 0; i < len(backoffs); i++ {
		backoffs[i] = base
		base *= 2
	}
	return backoffs
}

type retryer struct {
	backoff []time.Duration
	jitter  float64
}

func (r *retryer) SetBackoff(backoff []time.Duration) {
	r.backoff = backoff
}

// newRetryer constructs a retryer with the given backoff pattern and classifier. The length of the backoff pattern
// indicates how many times an action will be retried, and the value at each index indicates the amount of time
// waited before each subsequent retry. The classifier is used to determine which errors should be retried and
// which should cause the retrier to fail fast. The DefaultClassifier is used if nil is passed.
func newRetryer(backoff []time.Duration) *retryer {
	return &retryer{
		backoff: backoff,
	}
}

func worthToRetry(ctx context.Context, waitTime time.Duration) bool {
	if ctx == nil {
		return true
	}
	if ctx.Err() != nil {
		return false
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return true
	}
	now := time.Now()
	return now.UnixNano()+int64(waitTime) <= deadline.UnixNano()
}

// Run executes the given work function, then classifies its return value based on the classifier.
// If the result is Succeed or Fail, the return value of the work function is
// returned to the caller. If the result is Retry, then Run sleeps according to its backoff policy
// before retrying. If the total number of retries is exceeded then the return value of the work function
// is returned to the caller regardless.
func (r *retryer) Run(ctx context.Context, work func() error, classifier classifier) error {
	// run
	ferr := work()
	// try retry
	for i := 0; i < len(r.backoff) && classifier.Classify(ferr) == Retry; i++ {
		// 重试
		sleepTime := r.calcSleep(i)
		if !worthToRetry(ctx, sleepTime) {
			return ferr
		}
		time.Sleep(sleepTime)
		ferr = work()
	}
	return ferr
}

func (r *retryer) calcSleep(i int) time.Duration {
	// take a random float in the range (-r.jitter, +r.jitter) and multiply it by the base amount
	return r.backoff[i]
}

// SetJitter sets the amount of jitter on each back-off to a factor between 0.0 and 1.0 (values outside this range
// are silently ignored). When a retry occurs, the back-off is adjusted by a random amount up to this value.
func (r *retryer) SetJitter(jit float64) {
	if jit < 0 || jit > 1 {
		return
	}
	r.jitter = jit
}

// readCloserWithCRC warp io.ReadCloser with crc checker
type readCloserWithCRC struct {
	serverCrc uint64 // Get Object 时对 content 进行校验
	checker   hash.Hash64
	base      io.ReadCloser
}

func (r *readCloserWithCRC) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := r.base.(io.Seeker)
	if !ok {
		return 0, NotSupportSeek
	}

	if whence != io.SeekCurrent {
		r.checker.Reset()
	}

	return seeker.Seek(offset, whence)
}

func (r *readCloserWithCRC) Read(p []byte) (n int, err error) {
	n, err = r.base.Read(p)
	if n > 0 {
		if n, err = r.checker.Write(p[:n]); err != nil {
			return n, err
		}
	}
	if err == io.EOF && r.serverCrc != 0 {
		clientCRC := r.checker.Sum64()
		if clientCRC != r.serverCrc {
			return n, CrcCheckFail.withCause(fmt.Errorf("expect crc: %d , actual crc:%d", r.serverCrc, clientCRC))
		}
	}

	return
}

func (r *readCloserWithCRC) Close() error {
	return r.base.Close()
}

// parallelReadCloserWithListener warp multiple io.ReadCloser will be R/W in parallel with a same DataTransferListener
type parallelReadCloserWithListener struct {
	listener DataTransferListener
	base     io.ReadCloser
	consumed *int64
	subtotal *int64
	total    int64
}

func (r *parallelReadCloserWithListener) Read(p []byte) (n int, err error) {
	n, err = r.base.Read(p)
	if err != nil && err != io.EOF {
		postDataTransferStatus(r.listener, &DataTransferStatus{
			Type: enum.DataTransferFailed,
		})
		return n, err
	}
	if n <= 0 {
		return
	}
	subtotal := atomic.AddInt64(r.subtotal, int64(n))
	consumed := atomic.AddInt64(r.consumed, int64(n))

	if subtotal >= DefaultProgressCallbackSize && atomic.CompareAndSwapInt64(r.subtotal, subtotal, 0) {
		postDataTransferStatus(r.listener, &DataTransferStatus{
			Type:          enum.DataTransferRW,
			RWOnceBytes:   subtotal,
			ConsumedBytes: consumed,
			TotalBytes:    r.total,
		})
	}

	if consumed == r.total {
		for subtotal != 0 {
			if atomic.CompareAndSwapInt64(r.subtotal, subtotal, 0) {
				postDataTransferStatus(r.listener, &DataTransferStatus{
					Type:          enum.DataTransferRW,
					RWOnceBytes:   subtotal,
					ConsumedBytes: consumed,
					TotalBytes:    r.total,
				})
				break
			} else {
				subtotal = atomic.LoadInt64(r.subtotal)
			}
		}
		postDataTransferStatus(r.listener, &DataTransferStatus{
			Type:          enum.DataTransferSucceed,
			ConsumedBytes: consumed,
			TotalBytes:    r.total,
		})
	}
	return
}

func (r *parallelReadCloserWithListener) Close() error {
	return r.base.Close()
}

// readCloserWithListener warp io.ReadCloser with DataTransferListener
type readCloserWithListener struct {
	listener DataTransferListener
	base     io.ReadCloser
	consumed int64
	subtotal int64
	total    int64
	onceEof  bool
}

func (r *readCloserWithListener) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := r.base.(io.Seeker)
	if !ok {
		return 0, NotSupportSeek
	}
	if whence != io.SeekCurrent {
		r.consumed = 0
		r.subtotal = 0
		r.onceEof = false
	}

	return seeker.Seek(offset, whence)
}

func (r *readCloserWithListener) Read(p []byte) (n int, err error) {
	if r.consumed == 0 {
		postDataTransferStatus(r.listener, &DataTransferStatus{
			Type: enum.DataTransferStarted,
		})
	}
	defer func() {
		if err == io.EOF {
			r.consumed += int64(n)
			r.subtotal += int64(n)
			if r.subtotal != 0 {
				postDataTransferStatus(r.listener, &DataTransferStatus{
					Type:          enum.DataTransferRW,
					RWOnceBytes:   r.subtotal,
					ConsumedBytes: r.consumed,
					TotalBytes:    r.total,
				})
				r.subtotal = 0
			}

			if !r.onceEof {
				if r.total == -1 {
					r.total = r.consumed
				}
				postDataTransferStatus(r.listener, &DataTransferStatus{
					Type:          enum.DataTransferSucceed,
					ConsumedBytes: r.consumed,
					TotalBytes:    r.total,
				})
				r.onceEof = true
			}

		}
	}()
	n, err = r.base.Read(p)
	if err != nil && err != io.EOF {
		postDataTransferStatus(r.listener, &DataTransferStatus{
			Type: enum.DataTransferFailed,
		})
		return n, err
	}
	if n <= 0 || err == io.EOF {
		return
	}
	r.consumed += int64(n)
	r.subtotal += int64(n)
	if r.subtotal >= DefaultProgressCallbackSize {
		postDataTransferStatus(r.listener, &DataTransferStatus{
			Type:          enum.DataTransferRW,
			RWOnceBytes:   r.subtotal,
			ConsumedBytes: r.consumed,
			TotalBytes:    r.total,
		})
		r.subtotal = 0
	}
	return
}

func (r *readCloserWithListener) Close() error {
	return r.base.Close()
}

// ReadCloserWithLimiter warp io.ReadCloser with DataTransferListener
type ReadCloserWithLimiter struct {
	limiter  RateLimiter
	acquireN int
	base     io.ReadCloser
}

func (r *ReadCloserWithLimiter) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := r.base.(io.Seeker)
	if !ok {
		return 0, NotSupportSeek
	}
	r.acquireN = 0
	return seeker.Seek(offset, whence)
}

func (r *ReadCloserWithLimiter) Read(p []byte) (n int, err error) {
	want := len(p)
	if want > r.acquireN {
		// 需要申请的配额
		want = want - r.acquireN
		for {
			ok, timeToWait := r.limiter.Acquire(int64(want))
			if ok {
				break
			}
			time.Sleep(timeToWait)
		}
		r.acquireN += want
	}
	n, err = r.base.Read(p)
	// 实际消耗的配额
	r.acquireN = r.acquireN - n
	return n, err

}

func (r *ReadCloserWithLimiter) Close() error {
	return r.base.Close()
}

type copyTask struct {
	cli        *ClientV2
	input      *ResumableCopyObjectInput
	ctx        context.Context
	UploadID   string
	ContentMD5 string
	PartNumber int64
	Offset     uint64
	PartSize   int64
	PartInfo   copyPartInfo
}

func (c *copyTask) do() (interface{}, error) {
	uploadInput, ok := c.getBaseInput().(UploadPartV2Input)
	if ok {
		part, err := c.cli.UploadPartV2(c.ctx, &uploadInput)
		if err != nil {
			return nil, err
		}
		return copyPartInfo{PartNumber: int64(part.PartNumber), Etag: part.ETag, IsCompleted: true}, nil
	}
	input := c.getBaseInput().(UploadPartCopyV2Input)
	output, err := c.cli.UploadPartCopyV2(c.ctx, &input)
	if err != nil {
		return nil, err
	}
	return copyPartInfo{
		PartNumber:      int64(output.PartNumber),
		CopySourceRange: input.CopySourceRange,
		Etag:            output.ETag,
		IsCompleted:     true,
	}, nil

}

func (c *copyTask) getBaseInput() interface{} {
	if c.PartInfo.IsZeroSize {
		return UploadPartV2Input{UploadPartBasicInput: UploadPartBasicInput{
			Bucket:        c.input.Bucket,
			Key:           c.input.Key,
			UploadID:      c.UploadID,
			PartNumber:    1,
			SSECAlgorithm: c.input.SSECAlgorithm,
			SSECKey:       c.input.SSECKey,
			SSECKeyMD5:    c.input.SSECKeyMD5,
		}}
	}
	return UploadPartCopyV2Input{
		Bucket:                      c.input.Bucket,
		Key:                         c.input.Key,
		UploadID:                    c.UploadID,
		PartNumber:                  int(c.PartNumber),
		SrcBucket:                   c.input.SrcBucket,
		SrcKey:                      c.input.SrcKey,
		SrcVersionID:                c.input.SrcVersionID,
		CopySourceRangeStart:        c.PartInfo.CopySourceRangeStart,
		CopySourceRangeEnd:          c.PartInfo.CopySourceRangeEnd,
		CopySourceRange:             c.PartInfo.CopySourceRange,
		CopySourceIfMatch:           c.input.CopySourceIfMatch,
		CopySourceIfModifiedSince:   c.input.CopySourceIfModifiedSince,
		CopySourceIfNoneMatch:       c.input.CopySourceIfNoneMatch,
		CopySourceIfUnmodifiedSince: c.input.CopySourceIfUnmodifiedSince,
		CopySourceSSECAlgorithm:     c.input.CopySourceSSECAlgorithm,
		CopySourceSSECKey:           c.input.CopySourceSSECKey,
		CopySourceSSECKeyMD5:        c.input.CopySourceSSECKeyMD5,
		SSECKey:                     c.input.SSECKey,
		SSECKeyMD5:                  c.input.SSECKeyMD5,
		SSECAlgorithm:               c.input.SSECAlgorithm,
		TrafficLimit:                c.input.TrafficLimit,
	}
}
