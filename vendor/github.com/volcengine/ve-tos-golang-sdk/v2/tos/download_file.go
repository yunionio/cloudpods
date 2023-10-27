package tos

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

func getDownloadCheckpoint(input *DownloadFileInput, init func(input *HeadObjectV2Output) (*downloadCheckpoint, error), output *HeadObjectV2Output) (checkpoint *downloadCheckpoint, err error) {
	enabled := input.EnableCheckpoint
	checkpointPath := input.CheckpointFile
	if !enabled {
		return init(output)
	}

	checkpoint = &downloadCheckpoint{}
	loadCheckPoint(checkpointPath, checkpoint)
	if checkpoint.Valid(input, output) {
		return
	}

	parentDir := filepath.Dir(checkpointPath)
	stat, err := os.Stat(parentDir)
	if err != nil {
		err = os.MkdirAll(parentDir, os.ModePerm)
		if err != nil {
			return nil, newTosClientError(err.Error(), err)
		}
	} else if !stat.IsDir() {
		return nil, newTosClientError("Fail to create folder due to a same file exists.", nil)
	}

	file, err := os.Create(checkpointPath)
	if err != nil {
		return nil, newTosClientError(err.Error(), err)
	}
	_ = file.Close()

	checkpoint, err = init(output)
	if err != nil {
		return nil, err
	}

	err = checkpoint.WriteToFile()
	if err != nil {
		return nil, err
	}

	return
}

func (cli *ClientV2) DownloadFile(ctx context.Context, input *DownloadFileInput) (*DownloadFileOutput, error) {
	err := validateDownloadInput(input, cli.isCustomDomain)
	if err != nil {
		return nil, err
	}
	headOutput, err := cli.HeadObjectV2(ctx, &input.HeadObjectV2Input)
	if err != nil {
		return nil, err
	}

	needDownload, err := parseDownloadFilePath(input)
	if err != nil {
		return nil, err
	}
	if !needDownload {
		return &DownloadFileOutput{*headOutput}, nil
	}

	event := downloadEvent{input: input}
	init := func(output *HeadObjectV2Output) (*downloadCheckpoint, error) {
		err := createDownloadTempFile(input, event)
		if err != nil {
			return nil, err
		}
		return initDownloadCheckpoint(input, headOutput)
	}
	checkpoint, err := getDownloadCheckpoint(input, init, headOutput)
	if err != nil {
		return nil, err
	}
	cleaner := func() {
		_ = os.Remove(input.CheckpointFile)
		_ = os.Remove(input.tempFile)
	}
	bindCancelHookWithCleaner(input.CancelHook, cleaner)
	return cli.downloadFile(ctx, headOutput, checkpoint, input, event)
}

// loadCheckPoint load UploadFile checkpoint or DownloadFile checkpoint.
// checkpoint must be a pointer
func loadCheckPoint(path string, checkpoint interface{}) {
	contents, err := ioutil.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if len(contents) == 0 {
		return
	}
	json.Unmarshal(contents, &checkpoint)
}
func isDir(filePath string) bool {
	stat, err := os.Stat(filePath)
	if err != nil {
		_, fileName := filepath.Split(filePath)
		return fileName == ""
	}
	return stat.IsDir()
}

// if file is a directory, append suffix to it to make a file name
func withSuffixIfDir(filePath string, suffix string) string {
	if isDir(filePath) {
		return filepath.Clean(filepath.Join(filePath, suffix))
	}
	return filePath
}

func getDownloadCheckPointPath(checkpointPath, filePath, bucket, key, versionId string) string {
	fileName := strings.Join([]string{filepath.Base(filePath), checkpointPathMd5(bucket, key, versionId), "download"}, ".")
	if len(checkpointPath) == 0 {
		dirName := filepath.Dir(filePath)
		return filepath.Clean(filepath.Join(dirName, fileName))
	}
	return withSuffixIfDir(checkpointPath, fileName)

}

func checkpointPathMd5(bucket string, key string, versionId string) string {
	var data []byte
	if versionId != "" {
		data = []byte(strings.Join([]string{bucket, key, versionId}, "."))
	} else {
		data = []byte(strings.Join([]string{bucket, key}, "."))

	}
	r := md5.Sum(data)
	return base64.URLEncoding.EncodeToString(r[:])
}

func parseDownloadFilePath(input *DownloadFileInput) (needDownloadFile bool, err error) {
	input.filePath = input.FilePath
	inputFile := input.filePath
	isDirRes := isDir(input.filePath)
	if isDirRes {
		input.filePath = filepath.Clean(filepath.Join(input.filePath, input.Key))
	}
	input.tempFile = input.filePath + TempFileSuffix

	if input.EnableCheckpoint {
		input.CheckpointFile = getDownloadCheckPointPath(input.CheckpointFile, input.filePath, input.Bucket, input.Key, input.VersionID)
	}

	if isDirRes && strings.HasSuffix(input.Key, "/") {
		err := os.MkdirAll(filepath.Join(inputFile, input.Key), os.ModePerm)
		if err != nil {
			return false, InvalidFilePath.withCause(err)
		}
		return false, nil
	}

	return true, nil

}

func validateDownloadInput(input *DownloadFileInput, isCustomDomain bool) error {
	if err := isValidNames(input.Bucket, input.Key, isCustomDomain); err != nil {
		return err
	}

	if input.PartSize == 0 {
		input.PartSize = DefaultPartSize
	}
	if input.PartSize < MinPartSize || input.PartSize > MaxPartSize {
		return newTosClientError("The input part size is invalid, please set it range from 5MB to 5GB", nil)
	}

	if input.TaskNum < 1 {
		input.TaskNum = 1
	}
	if input.TaskNum > 1000 {
		input.TaskNum = 1000
	}
	return nil
}

func initDownloadCheckpoint(input *DownloadFileInput, headOutput *HeadObjectV2Output) (*downloadCheckpoint, error) {
	partsNum := headOutput.ContentLength / input.PartSize
	remainder := headOutput.ContentLength % input.PartSize
	if remainder != 0 {
		partsNum++
	}
	parts := make([]downloadPartInfo, partsNum)
	for i := int64(0); i < partsNum; i++ {
		parts[i] = downloadPartInfo{
			PartNumber: int(i + 1),
			RangeStart: i * input.PartSize,
			RangeEnd:   (i+1)*input.PartSize - 1,
		}
	}
	if remainder != 0 {
		parts[partsNum-1].RangeEnd = (partsNum-1)*input.PartSize + remainder - 1
	}
	if len(parts) > 10000 {
		return nil, newTosClientError("tos: part count too many", nil)
	}
	return &downloadCheckpoint{
		checkpointPath:    input.CheckpointFile,
		Bucket:            input.Bucket,
		Key:               input.Key,
		VersionID:         input.VersionID,
		PartSize:          input.PartSize,
		IfMatch:           input.IfMatch,
		IfModifiedSince:   input.IfModifiedSince,
		IfNoneMatch:       input.IfNoneMatch,
		IfUnmodifiedSince: input.IfUnmodifiedSince,
		SSECAlgorithm:     input.SSECAlgorithm,
		SSECKeyMD5:        input.SSECKey,
		ObjectInfo: objectInfo{
			Etag:          headOutput.ETag,
			HashCrc64ecma: headOutput.HashCrc64ecma,
			LastModified:  headOutput.LastModified,
			ObjectSize:    headOutput.ContentLength,
		},
		FileInfo: downloadFileInfo{
			FilePath:     input.filePath,
			TempFilePath: input.tempFile,
		},
		PartsInfo: parts,
	}, nil
}

func checkAndCreateDir(filePath string) error {
	dir := filepath.Dir(filePath)
	stat, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("dir name same as file name. ")
	}
	return nil
}

func createDownloadTempFile(input *DownloadFileInput, event downloadEvent) error {
	wrapErr := func(err error) error {
		event.postDownloadEvent(&DownloadEvent{
			Type:           enum.DownloadEventCreateTempFileFailed,
			Bucket:         input.Bucket,
			Key:            input.Key,
			VersionID:      input.VersionID,
			FilePath:       input.filePath,
			TempFilePath:   &input.tempFile,
			CheckpointFile: &input.CheckpointFile,
		})
		return newTosClientError("tos: create temp file failed.", err)
	}

	err := checkAndCreateDir(input.tempFile)
	if err != nil {
		return wrapErr(err)
	}
	file, err := os.Create(input.tempFile)
	if err != nil {
		return wrapErr(err)
	}
	_ = file.Close()
	event.postDownloadEvent(&DownloadEvent{
		Type:           enum.DownloadEventCreateTempFileSucceed,
		Bucket:         input.Bucket,
		Key:            input.Key,
		VersionID:      input.VersionID,
		FilePath:       input.filePath,
		TempFilePath:   &input.tempFile,
		CheckpointFile: &input.CheckpointFile,
	})
	return nil
}

func getDownloadTasks(cli *ClientV2, ctx context.Context, headOutput *HeadObjectV2Output,
	checkpoint *downloadCheckpoint, input *DownloadFileInput) []task {
	tasks := make([]task, 0)
	consumed := int64(0)
	subtotal := int64(0)
	for _, part := range checkpoint.PartsInfo {
		if !part.IsCompleted {
			tasks = append(tasks, &downloadTask{
				cli:         cli,
				ctx:         ctx,
				input:       input,
				partNumber:  part.PartNumber,
				rangeStart:  part.RangeStart,
				rangeEnd:    part.RangeEnd,
				consumed:    &consumed,
				subtotal:    &subtotal,
				total:       headOutput.ContentLength,
				enableCRC64: cli.enableCRC,
			})
		} else {
			consumed += part.RangeEnd - part.RangeStart + 1
		}
	}
	return tasks
}

func (d downloadEvent) newDownloadEvent() *DownloadEvent {
	return &DownloadEvent{
		Bucket:         d.input.Bucket,
		Key:            d.input.Key,
		VersionID:      d.input.VersionID,
		FilePath:       d.input.filePath,
		CheckpointFile: &d.input.CheckpointFile,
		TempFilePath:   &d.input.tempFile,
	}
}

func (d downloadEvent) newDownloadPartSucceedEvent(part downloadPartInfo) *DownloadEvent {
	event := d.newSucceedEvent(enum.DownloadEventDownloadPartSucceed)
	event.DowloadPartInfo = &DownloadPartInfo{
		PartNumber: part.PartNumber,
		RangeStart: part.RangeStart,
		RangeEnd:   part.RangeEnd,
	}
	return event
}

func (d downloadEvent) newSucceedEvent(eventType enum.DownloadEventType) *DownloadEvent {
	event := d.newDownloadEvent()
	event.Type = eventType
	return event
}

func (d downloadEvent) newFailedEvent(err error, eventType enum.DownloadEventType) *DownloadEvent {
	event := d.newDownloadEvent()
	event.Type = eventType
	event.Err = err
	return event
}

func (d downloadEvent) postDownloadEvent(event *DownloadEvent) {
	if d.input.DownloadEventListener != nil {
		d.input.DownloadEventListener.EventChange(event)
	}
}

func (cli *ClientV2) downloadFile(ctx context.Context,
	headOutput *HeadObjectV2Output, checkpoint *downloadCheckpoint, input *DownloadFileInput, event downloadEvent) (*DownloadFileOutput, error) {
	// prepare tasks
	tasks := getDownloadTasks(cli, ctx, headOutput, checkpoint, input)
	routinesNum := min(input.TaskNum, len(tasks))
	tg := newTaskGroup(getCancelHandle(input.CancelHook), routinesNum, checkpoint, event, input.EnableCheckpoint, tasks)
	tg.RunWorker()
	// start adding tasks
	postDataTransferStatus(input.DataTransferListener, &DataTransferStatus{
		Type: enum.DataTransferStarted,
	})
	tg.Scheduler()
	success, err := tg.Wait()
	if err != nil {
		_ = os.Remove(input.tempFile)
	}

	if success < len(tasks) {
		return nil, newTosClientError("tos: some download task failed.", nil)
	}
	// Check CRC64
	if cli.enableCRC && headOutput.HashCrc64ecma != 0 && combineCRCInDownload(checkpoint.PartsInfo) != headOutput.HashCrc64ecma {
		return nil, newTosClientError("tos: crc of entire file mismatch.", nil)
	}
	err = os.Rename(input.tempFile, input.filePath)
	if err != nil {
		event.postDownloadEvent(event.newFailedEvent(err, enum.DownloadEventRenameTempFileFailed))
		return nil, err
	}
	event.postDownloadEvent(event.newSucceedEvent(enum.DownloadEventRenameTempFileSucceed))
	_ = os.Remove(checkpoint.checkpointPath)
	return &DownloadFileOutput{*headOutput}, nil
}
