// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/block/bitmap"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/common"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/diskstream"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/footer"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/validator"

	"yunion.io/x/cloudmux/pkg/multicloud/azure/concurrent"
	"yunion.io/x/cloudmux/pkg/multicloud/azure/progress"
)

// DiskUploadContext type describes VHD upload context, this includes the disk stream to read from, the ranges of
// the stream to read, the destination blob and it's container, the client to communicate with Azure storage and
// the number of parallel go-routines to use for upload.
//

type DataWithRange struct {
	Range *common.IndexRange
	Data  []byte
}

type DiskUploadContext struct {
	VhdStream             *diskstream.DiskStream    // The stream whose ranges needs to be uploaded
	AlreadyProcessedBytes int64                     // The size in bytes already uploaded
	UploadableRanges      []*common.IndexRange      // The subset of stream ranges to be uploaded
	BlobServiceClient     storage.BlobStorageClient // The client to make Azure blob service API calls
	ContainerName         string                    // The container in which page blob resides
	BlobName              string                    // The destination page blob name
	Parallelism           int                       // The number of concurrent goroutines to be used for upload
	Resume                bool                      // Indicate whether this is a new or resuming upload
	MD5Hash               []byte                    // MD5Hash to be set in the page blob properties once upload finishes
}

// oneMB is one MegaByte
//
const oneMB = float64(1048576)

// Upload uploads the disk ranges described by the parameter cxt, this parameter describes the disk stream to
// read from, the ranges of the stream to read, the destination blob and it's container, the client to communicate
// with Azure storage and the number of parallel go-routines to use for upload.
//
func Upload(cxt *DiskUploadContext, callback func(float32)) error {
	// Get the channel that contains stream of disk data to upload
	dataWithRangeChan, streamReadErrChan := GetDataWithRanges(cxt.VhdStream, cxt.UploadableRanges)

	// The channel to send upload request to load-balancer
	requtestChan := make(chan *concurrent.Request, 0)

	// Prepare and start the load-balancer that load request across 'cxt.Parallelism' workers
	loadBalancer := concurrent.NewBalancer(cxt.Parallelism)
	loadBalancer.Init()
	workerErrorChan, allWorkersFinishedChan := loadBalancer.Run(requtestChan)

	// Calculate the actual size of the data to upload
	uploadSizeInBytes := int64(0)
	for _, r := range cxt.UploadableRanges {
		uploadSizeInBytes += r.Length()
	}
	fmt.Printf("\nEffective upload size: %.2f MB (from %.2f MB originally)", float64(uploadSizeInBytes)/oneMB, float64(cxt.VhdStream.GetSize())/oneMB)

	// Prepare and start the upload progress tracker
	uploadProgress := progress.NewStatus(cxt.Parallelism, cxt.AlreadyProcessedBytes, uploadSizeInBytes, progress.NewComputestateDefaultSize())
	progressChan := uploadProgress.Run()

	// read progress status from progress tracker and print it
	go readAndPrintProgress(progressChan, cxt.Resume, callback)

	// listen for errors reported by workers and print it
	var allWorkSucceeded = true
	go func() {
		for {
			fmt.Println(<-workerErrorChan)
			allWorkSucceeded = false
		}
	}()

	var err error
L:
	for {
		select {
		case dataWithRange, ok := <-dataWithRangeChan:
			if !ok {
				close(requtestChan)
				break L
			}

			// Create work request
			//
			containerClinet := cxt.BlobServiceClient.GetContainerReference(cxt.ContainerName)
			blobClient := containerClinet.GetBlobReference(cxt.BlobName)
			req := &concurrent.Request{
				Work: func() error {
					err := blobClient.WriteRange(
						storage.BlobRange{Start: uint64(dataWithRange.Range.Start), End: uint64(dataWithRange.Range.End)},
						bytes.NewReader(dataWithRange.Data),
						&storage.PutPageOptions{},
					)
					if err == nil {
						uploadProgress.ReportBytesProcessedCount(dataWithRange.Range.Length())
					}
					return err
				},
				ShouldRetry: func(e error) bool {
					return true
				},
				ID: dataWithRange.Range.String(),
			}

			// Send work request to load balancer for processing
			//
			requtestChan <- req
		case err = <-streamReadErrChan:
			close(requtestChan)
			loadBalancer.TearDownWorkers()
			break L
		}
	}

	<-allWorkersFinishedChan
	uploadProgress.Close()

	if !allWorkSucceeded {
		err = errors.New("\nUpload Incomplete: Some blocks of the VHD failed to upload, rerun the command to upload those blocks")
	}

	if err == nil {
		fmt.Printf("\r Completed: %3d%% [%10.2f MB] RemainingTime: %02dh:%02dm:%02ds Throughput: %d Mb/sec  %2c ",
			100,
			float64(uploadSizeInBytes)/oneMB,
			0, 0, 0,
			0, ' ')

	}
	return err
}

// GetDataWithRanges with start reading and streaming the ranges from the disk identified by the parameter ranges.
// It returns two channels, a data channel to stream the disk ranges and a channel to send any error while reading
// the disk. On successful completion the data channel will be closed. the caller must not expect any more value in
// the data channel if the error channel is signaled.
//
func GetDataWithRanges(stream *diskstream.DiskStream, ranges []*common.IndexRange) (<-chan *DataWithRange, <-chan error) {
	dataWithRangeChan := make(chan *DataWithRange, 0)
	errorChan := make(chan error, 0)
	go func() {
		for _, r := range ranges {
			dataWithRange := &DataWithRange{
				Range: r,
				Data:  make([]byte, r.Length()),
			}
			_, err := stream.Seek(r.Start, 0)
			if err != nil {
				errorChan <- err
				return
			}
			_, err = io.ReadFull(stream, dataWithRange.Data)
			if err != nil {
				errorChan <- err
				return
			}
			dataWithRangeChan <- dataWithRange
		}
		close(dataWithRangeChan)
	}()
	return dataWithRangeChan, errorChan
}

// readAndPrintProgress reads the progress records from the given progress channel and output it. It reads the
// progress record until the channel is closed.
//
func readAndPrintProgress(progressChan <-chan *progress.Record, resume bool, callback func(float32)) {
	var spinChars = [4]rune{'\\', '|', '/', '-'}
	s := time.Time{}
	if resume {
		fmt.Println("\nResuming VHD upload..")
	} else {
		fmt.Println("\nUploading the VHD..")
	}

	i := 0
	for progressRecord := range progressChan {
		if i == 4 {
			i = 0
		}
		t := s.Add(progressRecord.RemainingDuration)
		fmt.Printf("\r Completed: %3d%% [%10.2f MB] RemainingTime: %02dh:%02dm:%02ds Throughput: %d Mb/sec  %2c ",
			int(progressRecord.PercentComplete),
			float64(progressRecord.BytesProcessed)/oneMB,
			t.Hour(), t.Minute(), t.Second(),
			int(progressRecord.AverageThroughputMbPerSecond),
			spinChars[i],
		)
		if callback != nil {
			callback(33.0 + float32(progressRecord.PercentComplete*0.33))
		}
		i++
	}
}

func ensureVHDSanity(localVHDPath string) error {
	if err := validator.ValidateVhd(localVHDPath); err != nil {
		return err
	}

	if err := validator.ValidateVhdSize(localVHDPath); err != nil {
		return err
	}
	return nil
}

func LocateUploadableRanges(stream *diskstream.DiskStream, rangesToSkip []*common.IndexRange, pageSizeInBytes int64) ([]*common.IndexRange, error) {
	var err error
	var diskRanges = make([]*common.IndexRange, 0)
	stream.EnumerateExtents(func(ext *diskstream.StreamExtent, extErr error) bool {
		if extErr != nil {
			err = extErr
			return false
		}

		diskRanges = append(diskRanges, ext.Range)
		return true
	})

	if err != nil {
		return nil, err
	}

	diskRanges = common.SubtractRanges(diskRanges, rangesToSkip)
	diskRanges = common.ChunkRangesBySize(diskRanges, pageSizeInBytes)
	return diskRanges, nil
}

func DetectEmptyRanges(diskStream *diskstream.DiskStream, uploadableRanges []*common.IndexRange) ([]*common.IndexRange, error) {
	if diskStream.GetDiskType() != footer.DiskTypeFixed {
		return uploadableRanges, nil
	}

	fmt.Println("\nDetecting empty ranges..")
	totalRangesCount := len(uploadableRanges)
	lastIndex := int32(-1)
	emptyRangesCount := int32(0)
	bits := make([]byte, int32(math.Ceil(float64(totalRangesCount)/float64(8))))
	bmap := bitmap.NewBitMapFromByteSliceCopy(bits)
	indexChan, errChan := LocateNonEmptyRangeIndices(diskStream, uploadableRanges)
L:
	for {
		select {
		case index, ok := <-indexChan:
			if !ok {
				break L
			}
			bmap.Set(index, true)
			emptyRangesCount += index - lastIndex - 1
			lastIndex = index
			fmt.Printf("\r Empty ranges : %d/%d", emptyRangesCount, totalRangesCount)
		case err := <-errChan:
			return nil, err
		}
	}

	// Remove empty ranges from the uploadable ranges slice.
	i := int32(0)
	for j := 0; j < totalRangesCount; j++ {
		if set, _ := bmap.Get(int32(j)); set {
			uploadableRanges[i] = uploadableRanges[j]
			i++
		}
	}
	uploadableRanges = uploadableRanges[:i]
	return uploadableRanges, nil
}

func LocateNonEmptyRangeIndices(stream *diskstream.DiskStream, ranges []*common.IndexRange) (<-chan int32, <-chan error) {
	indexChan := make(chan int32, 0)
	errorChan := make(chan error, 0)
	go func() {
		count := int64(-1)
		var buf []byte
		for index, r := range ranges {
			if count != r.Length() {
				count = r.Length()
				buf = make([]byte, count)
			}

			_, err := stream.Seek(r.Start, 0)
			if err != nil {
				errorChan <- err
				return
			}
			_, err = io.ReadFull(stream, buf)
			if err != nil {
				errorChan <- err
				return
			}
			if !isAllZero(buf) {
				indexChan <- int32(index)
			}
		}
		close(indexChan)
	}()
	return indexChan, errorChan
}

// isAllZero returns true if the given byte slice contain all zeros
//
func isAllZero(buf []byte) bool {
	l := len(buf)
	j := 0
	for ; j < l; j++ {
		if buf[j] != byte(0) {
			break
		}
	}
	return j == l
}
