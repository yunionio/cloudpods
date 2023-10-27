package tos

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

// initUploadPartsInfo initialize parts info from file stat,return TosClientError if failed
func initUploadPartsInfo(uploadFileStat os.FileInfo, partSize int64) ([]uploadPartInfo, error) {
	partCount := uploadFileStat.Size() / partSize
	lastPartSize := uploadFileStat.Size() % partSize
	if lastPartSize != 0 {
		partCount++
	}
	if partCount > 10000 {
		return nil, InvalidFilePartNum
	}
	parts := make([]uploadPartInfo, 0, partCount)
	for i := int64(0); i < partCount; i++ {
		part := uploadPartInfo{
			PartNumber: int(i + 1),
			PartSize:   partSize,
			Offset:     uint64(i * partSize),
		}
		parts = append(parts, part)
	}
	if lastPartSize != 0 {
		parts[partCount-1].PartSize = lastPartSize
	}
	if uploadFileStat.Size() == 0 {
		parts = append(parts, uploadPartInfo{PartNumber: 1, PartSize: 0, Offset: 0})
	}
	return parts, nil
}

// initUploadCheckpoint initialize checkpoint file, return TosClientError if failed
func initUploadCheckpoint(input *UploadFileInput, stat os.FileInfo) (*uploadCheckpoint, error) {
	parts, err := initUploadPartsInfo(stat, input.PartSize)
	if err != nil {
		return nil, err
	}
	checkPoint := &uploadCheckpoint{
		checkpointPath: input.CheckpointFile,
		PartsInfo:      parts,
		Bucket:         input.Bucket,
		Key:            input.Key,
		PartSize:       input.PartSize,
		SSECAlgorithm:  input.SSECAlgorithm,
		SSECKeyMD5:     input.SSECKeyMD5,
		EncodingType:   input.ContentEncoding,
		FilePath:       input.FilePath,
		FileInfo: fileInfo{
			Size:         stat.Size(),
			LastModified: stat.ModTime().Unix(),
		},
	}
	return checkPoint, nil
}

func getUploadCheckpointFilePath(checkpointPath, filePath string, bucket, key string) string {
	fileName := strings.Join([]string{filepath.Base(filePath), checkpointPathMd5(bucket, key, ""), "upload"}, ".")
	if len(checkpointPath) == 0 {
		dirName := filepath.Dir(filePath)
		return filepath.Join(dirName, fileName)
	}

	return withSuffixIfDir(checkpointPath, fileName)
}

// validateUploadInput validate upload input, return TosClientError failed
func validateUploadInput(input *UploadFileInput, stat os.FileInfo, isCustomDomain bool) error {
	if err := isValidNames(input.Bucket, input.Key, isCustomDomain); err != nil {
		return err
	}
	if input.PartSize == 0 {
		input.PartSize = DefaultPartSize
	}
	if input.PartSize < MinPartSize || input.PartSize > MaxPartSize {
		return InvalidPartSize
	}

	if stat.IsDir() {
		return newTosClientError("tos: does not support directory, please specific your file path.", nil)
	}
	if input.EnableCheckpoint {
		// get correct checkpoint path
		input.CheckpointFile = getUploadCheckpointFilePath(input.CheckpointFile, input.FilePath, input.Bucket, input.Key)
	}
	if input.TaskNum < 1 {
		input.TaskNum = 1
	}
	if input.TaskNum > 1000 {
		input.TaskNum = 1000
	}
	return nil
}

func (u *uploadPostEvent) postUploadEvent(event *UploadEvent) {
	if u.input.UploadEventListener != nil {
		u.input.UploadEventListener.EventChange(event)
	}
}

func loadExistUploadCheckPoint(ctx context.Context, cli *ClientV2, input *UploadFileInput, srcFile os.FileInfo) (*uploadCheckpoint, bool) {
	checkpoint := &uploadCheckpoint{}
	var err error
	loadCheckPoint(input.CheckpointFile, checkpoint)
	if checkpoint.Valid(srcFile, input.Bucket, input.Key, input.FilePath) {
		return checkpoint, true
	} else if checkpoint.Bucket != "" && checkpoint.Key != "" && checkpoint.UploadID != "" {
		// 尝试去 abort
		_, err = cli.AbortMultipartUpload(ctx,
			&AbortMultipartUploadInput{
				Bucket:   checkpoint.Bucket,
				Key:      checkpoint.Key,
				UploadID: checkpoint.UploadID})
		if err != nil && cli.logger != nil {
			cli.logger.Debug("fail to abort upload task: %s, err:%s", checkpoint.UploadID, err.Error())
		}
	}
	return nil, false
}

// getUploadCheckpoint get struct checkpoint from checkpoint file if checkpointPath is valid,
// or initialize from scratch with function init
func getUploadCheckpoint(ctx context.Context, cli *ClientV2, input *UploadFileInput, srcFile os.FileInfo, init func() (*uploadCheckpoint, error)) (checkpoint *uploadCheckpoint, err error) {

	if !input.EnableCheckpoint {
		return init()
	}

	checkpoint, exist := loadExistUploadCheckPoint(ctx, cli, input, srcFile)

	if exist {
		return checkpoint, nil
	}
	err = checkAndCreateDir(input.CheckpointFile)
	if err != nil {
		return nil, InvalidCheckpointFilePath.withCause(err)
	}

	file, err := os.Create(input.CheckpointFile)
	if err != nil {
		return nil, newTosClientError("tos: create checkpoint file failed", err)
	}
	_ = file.Close()

	checkpoint, err = init()
	if err != nil {
		return nil, err
	}

	err = checkpoint.WriteToFile()
	if err != nil {
		return nil, err
	}

	return
}

func bindCancelHookWithAborter(hook CancelHook, aborter func() error) {
	if hook == nil {
		return
	}
	cancel := hook.(*canceler)
	cancel.aborter = aborter
}

func bindCancelHookWithCleaner(hook CancelHook, cleaner func()) {
	if hook == nil {
		return
	}
	cancel := hook.(*canceler)
	cancel.cleaner = cleaner
}

func (cli *ClientV2) UploadFile(ctx context.Context, input *UploadFileInput) (output *UploadFileOutput, err error) {
	// avoid modifying on origin pointer
	input = &(*input)

	stat, err := os.Stat(input.FilePath)
	if err != nil {
		return nil, InvalidSrcFilePath
	}

	if err = validateUploadInput(input, stat, cli.isCustomDomain); err != nil {
		return nil, err
	}

	init := func() (*uploadCheckpoint, error) {
		return initUploadCheckpoint(input, stat)
	}

	// if the checkpoint file not exist, here we will create it
	checkpoint, err := getUploadCheckpoint(ctx, cli, input, stat, init)
	if err != nil {
		return nil, err
	}

	event := &uploadPostEvent{
		input:      input,
		checkPoint: checkpoint,
	}
	if checkpoint.UploadID == "" {
		// create multipart upload task
		created, err := cli.CreateMultipartUploadV2(ctx, &input.CreateMultipartUploadV2Input)
		if err != nil {
			event.postUploadEvent(&UploadEvent{
				Type:           enum.UploadEventCreateMultipartUploadFailed,
				Err:            err,
				Bucket:         input.Bucket,
				Key:            input.Key,
				CheckpointFile: &input.CheckpointFile,
			})
			return nil, err
		}
		event.postUploadEvent(&UploadEvent{
			Type:           enum.UploadEventCreateMultipartUploadSucceed,
			Bucket:         input.Bucket,
			Key:            input.Key,
			UploadID:       &created.UploadID,
			CheckpointFile: &input.CheckpointFile,
		})
		checkpoint.UploadID = created.UploadID
	}

	cleaner := func() {
		_ = os.Remove(input.CheckpointFile)
	}
	event.checkPoint = checkpoint
	bindCancelHookWithCleaner(input.CancelHook, cleaner)
	return cli.uploadPart(ctx, checkpoint, input, event)
}

func prepareUploadTasks(cli *ClientV2, ctx context.Context, checkpoint *uploadCheckpoint, input *UploadFileInput) []task {
	tasks := make([]task, 0)
	consumed := int64(0)
	subtotal := int64(0)
	for _, part := range checkpoint.PartsInfo {
		if !part.IsCompleted {
			tasks = append(tasks, &uploadTask{
				cli:        cli,
				ctx:        ctx,
				input:      input,
				total:      checkpoint.FileInfo.Size,
				UploadID:   checkpoint.UploadID,
				PartNumber: part.PartNumber,
				subtotal:   &subtotal,
				consumed:   &consumed,
				Offset:     part.Offset,
				PartSize:   part.PartSize,
			})
		} else {
			consumed += part.PartSize
		}
	}
	return tasks
}

func (u *uploadPostEvent) newUploadPartSucceedEvent(input *UploadFileInput, part uploadPartInfo) *UploadEvent {
	return &UploadEvent{
		Type:           enum.UploadEventUploadPartSucceed,
		Bucket:         input.Bucket,
		Key:            input.Key,
		UploadID:       part.uploadID,
		CheckpointFile: &input.CheckpointFile,
		UploadPartInfo: &UploadPartInfo{
			PartNumber:    part.PartNumber,
			PartSize:      part.PartSize,
			Offset:        int64(part.Offset),
			ETag:          &part.ETag,
			HashCrc64ecma: &part.HashCrc64ecma,
		},
	}
}

func (u *uploadPostEvent) newUploadPartAbortedEvent(input *UploadFileInput, uploadID string, err error) *UploadEvent {
	return &UploadEvent{
		Type:           enum.UploadEventUploadPartAborted,
		Err:            err,
		Bucket:         input.Bucket,
		Key:            input.Key,
		UploadID:       &uploadID,
		CheckpointFile: &input.CheckpointFile,
	}
}

func (u *uploadPostEvent) newUploadPartFailedEvent(input *UploadFileInput, uploadID string, err error) *UploadEvent {
	return &UploadEvent{
		Type:           enum.UploadEventUploadPartFailed,
		Err:            err,
		Bucket:         input.Bucket,
		Key:            input.Key,
		UploadID:       &uploadID,
		CheckpointFile: &input.CheckpointFile,
	}
}

func (u *uploadPostEvent) newCompleteMultipartUploadFailedEvent(input *UploadFileInput, uploadID string, err error) *UploadEvent {
	return &UploadEvent{
		Type:           enum.UploadEventCompleteMultipartUploadFailed,
		Err:            err,
		Bucket:         input.Bucket,
		Key:            input.Key,
		UploadID:       &uploadID,
		CheckpointFile: &input.CheckpointFile,
	}
}

func newCompleteMultipartUploadSucceedEvent(input *UploadFileInput, uploadID string) *UploadEvent {
	return &UploadEvent{
		Type:           enum.UploadEventCompleteMultipartUploadSucceed,
		Bucket:         input.Bucket,
		Key:            input.Key,
		UploadID:       &uploadID,
		CheckpointFile: &input.CheckpointFile,
	}
}

func postDataTransferStatus(listener DataTransferListener, status *DataTransferStatus) {
	if listener != nil {
		listener.DataTransferStatusChange(status)
	}
}

func getCancelHandle(hook CancelHook) chan struct{} {
	if c, ok := hook.(*canceler); ok {
		return c.cancelHandle
	}
	return make(chan struct{})
}

func combineCRCInDownload(parts []downloadPartInfo) uint64 {
	if len(parts) == 0 {
		return 0
	}
	crc := parts[0].HashCrc64ecma
	for i := 1; i < len(parts); i++ {
		crc = CRC64Combine(crc, parts[i].HashCrc64ecma, uint64(parts[i].RangeEnd-parts[i].RangeStart+1))
	}
	return crc
}

// combineCRCInParts calculates the total CRC of continuous parts
func combineCRCInParts(parts []uploadPartInfo) uint64 {
	if parts == nil || len(parts) == 0 {
		return 0
	}
	crc := parts[0].HashCrc64ecma
	for i := 1; i < len(parts); i++ {
		crc = CRC64Combine(crc, parts[i].HashCrc64ecma, uint64(parts[i].PartSize))
	}
	return crc
}

func (cli *ClientV2) uploadPart(ctx context.Context, checkpoint *uploadCheckpoint, input *UploadFileInput, event *uploadPostEvent) (*UploadFileOutput, error) {
	// prepare tasks
	// if amount of tasks >= 10000, err "tos: part count too many" will be raised.
	tasks := prepareUploadTasks(cli, ctx, checkpoint, input)
	routinesNum := min(input.TaskNum, len(tasks))
	cancelHandle := getCancelHandle(input.CancelHook)
	tg := newTaskGroup(cancelHandle, routinesNum, checkpoint, event, input.EnableCheckpoint, tasks)
	abort := func() error {
		_, err := cli.AbortMultipartUpload(ctx,
			&AbortMultipartUploadInput{
				Bucket:   input.Bucket,
				Key:      input.Key,
				UploadID: checkpoint.UploadID})
		_ = os.Remove(input.CheckpointFile)
		return err
	}
	bindCancelHookWithAborter(input.CancelHook, abort)

	tg.RunWorker()
	// start adding tasks
	postDataTransferStatus(input.DataTransferListener, &DataTransferStatus{
		TotalBytes: checkpoint.FileInfo.Size,
		Type:       enum.DataTransferStarted,
	})

	tg.Scheduler()
	success, taskErr := tg.Wait()
	if taskErr != nil {
		if err := abort(); err != nil {
			return nil, err
		}
		return nil, taskErr
	}
	// handle results
	if success < len(tasks) {
		return nil, newTosClientError("tos: some upload tasks failed.", nil)
	}
	complete, err := cli.CompleteMultipartUploadV2(ctx, &CompleteMultipartUploadV2Input{
		Bucket:   input.Bucket,
		Key:      input.Key,
		UploadID: checkpoint.UploadID,
		Parts:    checkpoint.GetParts(),
	})
	if err != nil {
		event.postUploadEvent(event.newCompleteMultipartUploadFailedEvent(input, checkpoint.UploadID, err))
		return nil, err
	}
	event.postUploadEvent(newCompleteMultipartUploadSucceedEvent(input, checkpoint.UploadID))

	if cli.enableCRC && complete.HashCrc64ecma != 0 && combineCRCInParts(checkpoint.PartsInfo) != complete.HashCrc64ecma {
		return nil, newTosClientError("tos: crc of entire file mismatch.", nil)

	}
	_ = os.Remove(input.CheckpointFile)

	return &UploadFileOutput{
		RequestInfo:   complete.RequestInfo,
		Bucket:        complete.Bucket,
		Key:           complete.Key,
		UploadID:      checkpoint.UploadID,
		ETag:          complete.ETag,
		Location:      complete.Location,
		VersionID:     complete.VersionID,
		HashCrc64ecma: complete.HashCrc64ecma,
		SSECAlgorithm: checkpoint.SSECAlgorithm,
		SSECKeyMD5:    checkpoint.SSECKeyMD5,
		EncodingType:  checkpoint.EncodingType,
	}, nil
}
