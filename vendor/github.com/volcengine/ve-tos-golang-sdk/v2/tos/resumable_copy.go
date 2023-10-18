package tos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

func parseResumableCopyObjectPath(input *ResumableCopyObjectInput) {
	isDirRes := isDir(input.CheckpointFile)
	if isDirRes || input.CheckpointFile == "" {
		input.CheckpointFile = filepath.Clean(filepath.Join(input.CheckpointFile, fmt.Sprintf("%s.%s.%s.%s.%s", input.SrcBucket, input.SrcKey, input.SrcVersionID, input.Bucket, input.Key)))
	}
}

func loadExistCopyCheckPoint(ctx context.Context, cli *ClientV2, input *ResumableCopyObjectInput, headOutput *HeadObjectV2Output) (*copyObjectCheckpoint, bool) {
	checkpoint := &copyObjectCheckpoint{}
	var err error
	loadCheckPoint(input.CheckpointFile, checkpoint)
	if checkpoint.Valid(input, headOutput) {
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

func getResumableCopyObjectCheckpoint(ctx context.Context, cli *ClientV2, input *ResumableCopyObjectInput, headOutput *HeadObjectV2Output, init func() (*copyObjectCheckpoint, error)) (checkpoint *copyObjectCheckpoint, err error) {
	if !input.EnableCheckpoint {
		return init()
	}

	parseResumableCopyObjectPath(input)

	checkpoint, exist := loadExistCopyCheckPoint(ctx, cli, input, headOutput)

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

func initCopyPartsInfo(headOutput *HeadObjectV2Output, partSize int64) ([]copyPartInfo, error) {
	if headOutput.ContentLength == 0 {
		return []copyPartInfo{{
			PartNumber: 1,
			IsZeroSize: true,
		}}, nil
	}
	partCount := headOutput.ContentLength / partSize
	remainder := headOutput.ContentLength % partSize
	if remainder != 0 {
		partCount++
	}
	if partCount > 10000 {
		return nil, InvalidFilePartNum
	}
	parts := make([]copyPartInfo, 0, partCount)
	for i := int64(0); i < partCount; i++ {
		part := copyPartInfo{
			PartNumber:           i + 1,
			CopySourceRangeStart: i * partSize,
			CopySourceRangeEnd:   (i+1)*partSize - 1,
			CopySourceRange:      fmt.Sprintf("bytes=%d-%d", i*partSize, (i+1)*partSize-1),
		}
		parts = append(parts, part)
	}
	if remainder != 0 {
		parts[partCount-1].CopySourceRangeEnd = (partCount-1)*partSize + remainder - 1
		parts[partCount-1].CopySourceRange = fmt.Sprintf("bytes=%d-%d", (partCount-1)*partSize, (partCount-1)*partSize+remainder-1)
	}
	return parts, nil
}

func initCopyCheckpoint(input *ResumableCopyObjectInput, headOutput *HeadObjectV2Output) (*copyObjectCheckpoint, error) {
	parts, err := initCopyPartsInfo(headOutput, input.PartSize)
	if err != nil {
		return nil, err
	}
	cp := &copyObjectCheckpoint{
		Bucket:                      input.Bucket,
		Key:                         input.Key,
		SrcBucket:                   input.SrcBucket,
		SrcVersionID:                input.SrcVersionID,
		PartSize:                    input.PartSize,
		UploadID:                    "",
		CopySourceIfMatch:           input.CopySourceIfMatch,
		CopySourceIfModifiedSince:   input.CopySourceIfModifiedSince,
		CopySourceIfNoneMatch:       input.CopySourceIfNoneMatch,
		CopySourceIfUnmodifiedSince: input.CopySourceIfUnmodifiedSince,
		CopySourceSSECAlgorithm:     input.CopySourceSSECAlgorithm,
		CopySourceSSECKeyMD5:        input.CopySourceSSECKeyMD5,
		SSECAlgorithm:               input.SSECAlgorithm,
		SSECKeyMD5:                  input.SSECKeyMD5,
		EncodingType:                input.EncodingType,
		CopySourceObjectInfo: objectInfo{
			Etag:          headOutput.ETag,
			HashCrc64ecma: headOutput.HashCrc64ecma,
			LastModified:  headOutput.LastModified,
			ObjectSize:    headOutput.ContentLength,
		},
		PartsInfo:      parts,
		CheckpointPath: input.CheckpointFile,
	}
	return cp, nil
}
func prepareCopyTasks(cli *ClientV2, ctx context.Context, checkpoint *copyObjectCheckpoint, input *ResumableCopyObjectInput) []task {
	tasks := make([]task, 0)
	for _, part := range checkpoint.PartsInfo {
		if !part.IsCompleted {
			tasks = append(tasks, &copyTask{
				cli:        cli,
				ctx:        ctx,
				input:      input,
				UploadID:   checkpoint.UploadID,
				PartNumber: part.PartNumber,
				PartInfo:   part,
			})
		}
	}
	return tasks
}

func (cli *ClientV2) copyPart(ctx context.Context, cp *copyObjectCheckpoint, input *ResumableCopyObjectInput, event *copyEvent) (*ResumableCopyObjectOutput, error) {
	tasks := prepareCopyTasks(cli, ctx, cp, input)
	routinesNum := min(input.TaskNum, len(tasks))
	cancelHandle := getCancelHandle(input.CancelHook)
	tg := newTaskGroup(cancelHandle, routinesNum, cp, event, input.EnableCheckpoint, tasks)
	abort := func() error {
		_, err := cli.AbortMultipartUpload(ctx,
			&AbortMultipartUploadInput{
				Bucket:   input.Bucket,
				Key:      input.Key,
				UploadID: cp.UploadID})
		return err
	}
	bindCancelHookWithAborter(input.CancelHook, abort)
	tg.RunWorker()
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
		return nil, newTosClientError("tos: some tasks copy failed.", nil)
	}
	complete, err := cli.CompleteMultipartUploadV2(ctx, &CompleteMultipartUploadV2Input{
		Bucket:   input.Bucket,
		Key:      input.Key,
		UploadID: cp.UploadID,
		Parts:    cp.GetParts(),
	})
	if err != nil {
		event.postCopyEvent(&CopyEvent{
			Type:           enum.CopyEventCompleteMultipartUploadFailed,
			Err:            err,
			Bucket:         input.Bucket,
			Key:            input.Key,
			UploadID:       &cp.UploadID,
			SrcBucket:      input.SrcBucket,
			SrcKey:         input.SrcKey,
			SrcVersionID:   input.SrcVersionID,
			CheckpointFile: &input.CheckpointFile,
		})
		return nil, err
	}
	event.postCopyEvent(&CopyEvent{
		Type:           enum.CopyEventCompleteMultipartUploadSucceed,
		Err:            err,
		Bucket:         input.Bucket,
		Key:            input.Key,
		UploadID:       &cp.UploadID,
		SrcBucket:      input.SrcBucket,
		SrcKey:         input.SrcKey,
		SrcVersionID:   input.SrcVersionID,
		CheckpointFile: &input.CheckpointFile,
	})
	_ = os.Remove(input.CheckpointFile)

	return &ResumableCopyObjectOutput{
		RequestInfo:   complete.RequestInfo,
		Bucket:        complete.Bucket,
		Key:           complete.Key,
		UploadID:      cp.UploadID,
		Etag:          complete.ETag,
		Location:      complete.Location,
		VersionID:     complete.VersionID,
		HashCrc64ecma: complete.HashCrc64ecma,
		SSECAlgorithm: cp.SSECAlgorithm,
		SSECKeyMD5:    cp.SSECKeyMD5,
		EncodingType:  cp.EncodingType,
	}, nil
}

func (cli *ClientV2) ResumableCopyObject(ctx context.Context, input *ResumableCopyObjectInput) (*ResumableCopyObjectOutput, error) {
	rawInput := *input
	copyInput := &rawInput
	headOutput, err := cli.HeadObjectV2(ctx, &HeadObjectV2Input{
		Bucket:            copyInput.SrcBucket,
		Key:               copyInput.SrcKey,
		VersionID:         copyInput.SrcVersionID,
		SSECAlgorithm:     copyInput.CopySourceSSECAlgorithm,
		SSECKey:           copyInput.CopySourceSSECKey,
		SSECKeyMD5:        copyInput.CopySourceSSECKeyMD5,
		IfModifiedSince:   copyInput.CopySourceIfModifiedSince,
		IfNoneMatch:       copyInput.CopySourceIfNoneMatch,
		IfUnmodifiedSince: copyInput.CopySourceIfUnmodifiedSince,
		IfMatch:           copyInput.CopySourceIfMatch,
	})
	if err != nil {
		return nil, err
	}
	event := &copyEvent{input: copyInput}
	init := func() (*copyObjectCheckpoint, error) {
		return initCopyCheckpoint(copyInput, headOutput)
	}
	cp, err := getResumableCopyObjectCheckpoint(ctx, cli, copyInput, headOutput, init)
	if err != nil {
		return nil, err
	}
	if cp.UploadID == "" {
		created, err := cli.CreateMultipartUploadV2(ctx, &copyInput.CreateMultipartUploadV2Input)
		if err != nil {
			event.postCopyEvent(&CopyEvent{
				Type:           enum.CopyEventCreateMultipartUploadFailed,
				Err:            err,
				Bucket:         copyInput.Bucket,
				Key:            copyInput.Key,
				SrcBucket:      copyInput.SrcBucket,
				SrcKey:         copyInput.SrcKey,
				SrcVersionID:   copyInput.SrcVersionID,
				CheckpointFile: &copyInput.CheckpointFile,
			})
			return nil, err
		}
		event.uploadID = created.UploadID
		event.postCopyEvent(&CopyEvent{
			Type:           enum.CopyEventCreateMultipartUploadSucceed,
			Bucket:         copyInput.Bucket,
			Key:            copyInput.Key,
			SrcBucket:      copyInput.SrcBucket,
			SrcKey:         copyInput.SrcKey,
			SrcVersionID:   copyInput.SrcVersionID,
			CheckpointFile: &copyInput.CheckpointFile,
		})
		cp.UploadID = created.UploadID
	}
	cleaner := func() {
		_ = os.Remove(copyInput.CheckpointFile)
	}
	bindCancelHookWithCleaner(copyInput.CancelHook, cleaner)

	return cli.copyPart(ctx, cp, copyInput, event)
}
