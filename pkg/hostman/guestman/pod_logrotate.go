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

package guestman

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/docker/go-units"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/options"
)

const (
	containerLogRotateInterval = 10 * time.Minute
)

var (
	containerLogRotateMu sync.Mutex
)

// RunContainerLogRotate runs log rotation for all running pod containers once.
// It is safe to call concurrently; only one run executes at a time.
func RunContainerLogRotate(ctx context.Context, manager *SGuestManager, maxSizeBytes int64, maxFiles int) {
	if maxSizeBytes <= 0 || maxFiles <= 0 {
		return
	}
	if !containerLogRotateMu.TryLock() {
		return
	}
	defer containerLogRotateMu.Unlock()

	cri := manager.host.GetCRI()
	if cri == nil {
		return
	}
	runtimeClient := cri.GetRuntimeClient()
	if runtimeClient == nil {
		return
	}

	manager.Servers.Range(func(_id, value interface{}) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		pod, ok := value.(PodInstance)
		if !ok {
			return true
		}
		if !pod.IsRunning() {
			return true
		}
		logDir := pod.GetPodLogDir()
		for ctrId, criId := range pod.ListContainerCriIds() {
			if criId == "" {
				continue
			}
			logPath := filepath.Join(logDir, pod.GetContainerLogPath(ctrId))
			if err := rotateContainerLog(ctx, logPath, criId, maxSizeBytes, maxFiles, runtimeClient); err != nil {
				log.Warningf("rotate container log %s (cri %s): %v", logPath, criId, err)
			}
		}
		return true
	})
}

// rotateContainerLog rotates the container log file at logPath if it exceeds maxSizeBytes,
// keeps up to maxFiles (current + rotated), then calls ReopenContainerLog for the container.
func rotateContainerLog(ctx context.Context, logPath, criId string, maxSizeBytes int64, maxFiles int, runtimeClient runtimeapi.RuntimeServiceClient) error {
	dir := filepath.Dir(logPath)
	base := filepath.Base(logPath)
	// Always try to cleanup stale rotated logs, even if we don't rotate this time.
	cleanupRotatedLogs(dir, base, maxFiles)

	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if info.Size() < maxSizeBytes {
		return nil
	}

	// Rename from high to low so we don't overwrite: .(n-1)->.n, ..., .1->.2, then main->.1
	for i := maxFiles - 1; i >= 2; i-- {
		src := filepath.Join(dir, base+"."+strconv.Itoa(i-1))
		dst := filepath.Join(dir, base+"."+strconv.Itoa(i))
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			log.Warningf("rename %s -> %s: %v", src, dst, err)
		}
	}
	// Then rotate current log to .1
	dst1 := filepath.Join(dir, base+".1")
	if err := os.Rename(logPath, dst1); err != nil {
		return errors.Wrapf(err, "rename %s -> %s", logPath, dst1)
	}
	// Cleanup again after shift.
	cleanupRotatedLogs(dir, base, maxFiles)

	_, err = runtimeClient.ReopenContainerLog(ctx, &runtimeapi.ReopenContainerLogRequest{
		ContainerId: criId,
	})
	if err != nil {
		// If runtime failed to reopen the log, try best to rename back so containerd keeps writing to logPath.
		if _, statErr := os.Stat(logPath); os.IsNotExist(statErr) {
			if rbErr := os.Rename(dst1, logPath); rbErr != nil && !os.IsNotExist(rbErr) {
				log.Warningf("reopen log failed, rename back %s -> %s: %v", dst1, logPath, rbErr)
			}
		}
		return errors.Wrap(err, "ReopenContainerLog")
	}
	return nil
}

func cleanupRotatedLogs(dir, base string, maxFiles int) {
	// Keep only .1 .. .(maxFiles-1). Remove .maxFiles and above.
	if maxFiles <= 0 {
		return
	}
	// Stop after some consecutive not-exist to avoid infinite loop.
	miss := 0
	for i := maxFiles; i < maxFiles+100; i++ {
		p := filepath.Join(dir, base+"."+strconv.Itoa(i))
		if err := os.Remove(p); err != nil {
			if os.IsNotExist(err) {
				miss++
				if miss >= 20 {
					return
				}
				continue
			}
			log.Errorf("remove old container log %s: %v", p, err)
			continue
		}
		log.Infof("remove old container log %s", p)
		miss = 0
	}
}

// StartContainerLogRotateLoop starts a goroutine that periodically runs container log rotation
// when options are enabled. Call from guestman after manager and host are ready.
func StartContainerLogRotateLoop(manager *SGuestManager) {
	maxSizeStr := options.HostOptions.ContainerLogMaxSize
	maxFiles := options.HostOptions.ContainerLogMaxFiles
	if maxSizeStr == "" || maxFiles <= 0 {
		return
	}
	maxSizeBytes, err := units.FromHumanSize(maxSizeStr)
	if err != nil {
		log.Warningf("parse ContainerLogMaxSize %q: %v, disable container log rotate", maxSizeStr, err)
		return
	}
	if maxSizeBytes <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(containerLogRotateInterval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*containerLogRotateInterval)
			RunContainerLogRotate(ctx, manager, maxSizeBytes, maxFiles)
			cancel()
		}
	}()
	log.Infof("container log rotate started: maxSize=%s, maxFiles=%d", maxSizeStr, maxFiles)
}
