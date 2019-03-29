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

package progress

import (
	"io"
	"time"
)

// ReaderWithProgress wraps an io.ReadCloser, it track and report the read progress.
//
type ReaderWithProgress struct {
	ProgressChan    <-chan *Record
	innerReadCloser io.ReadCloser
	progressStatus  *Status
}

// NewReaderWithProgress creates a new instance of ReaderWithProgress. The parameter inner is the inner stream whose
// read progress needs to be tracked, sizeInBytes is the total size of the inner stream in bytes,
// progressIntervalInSeconds is the interval at which the read progress needs to be send to ProgressChan channel.
// After using the this reader, it must be closed by calling Close method to avoid goroutine leak.
//
func NewReaderWithProgress(inner io.ReadCloser, sizeInBytes int64, progressIntervalInSeconds time.Duration) *ReaderWithProgress {
	r := &ReaderWithProgress{}
	r.innerReadCloser = inner
	r.progressStatus = NewStatus(0, 0, sizeInBytes, NewComputestateDefaultSize())
	r.ProgressChan = r.progressStatus.Run()
	return r
}

// Read reads up to len(b) bytes from the inner stream. It returns the number of bytes read and an error, if any.
// EOF is signaled when no more data to read and n will set to 0.
//
func (r *ReaderWithProgress) Read(p []byte) (n int, err error) {
	n, err = r.innerReadCloser.Read(p)
	if err == nil {
		r.progressStatus.ReportBytesProcessedCount(int64(n))
	}
	return
}

// Close closes the inner stream and stop reporting read progress in the ProgressChan chan.
//
func (r *ReaderWithProgress) Close() error {
	err := r.innerReadCloser.Close()
	r.progressStatus.Close()
	return err
}
