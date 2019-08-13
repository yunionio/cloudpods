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

package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type LogFileHook struct {
	FileDir  string
	FileName string
	fullPath string
	file     *os.File
	written  int64
}

func (h *LogFileHook) Init() error {
	if fi, err := os.Lstat(h.FileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(h.FileDir, 0755)
		} else {
			return fmt.Errorf("Lstat %s: %s", h.FileDir, err)
		}
	} else if !fi.Mode().IsDir() {
		return fmt.Errorf("%s exists and it's not a directory", h.FileDir)
	}

	h.fullPath = filepath.Join(h.FileDir, h.FileName)
	file, err := os.OpenFile(h.fullPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("OpenFile %s: %s", h.fullPath, err)
	}
	h.file = file
	h.written = 0
	return nil
}

func (h *LogFileHook) DeInit() {
	if h.file != nil {
		h.file.Close()
	}
}

func (h *LogFileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *LogFileHook) Fire(e *logrus.Entry) error {
	if b, err := e.Logger.Formatter.Format(e); err != nil {
		return err
	} else {
		n, err := h.file.Write(b)
		h.written += int64(n)
		return err
	}
	return nil
}

func (h *LogFileHook) Written() int64 {
	return h.written
}

// rotate by size
type LogFileRotateHook struct {
	LogFileHook
	RotateNum  int
	RotateSize int64
	filePaths  []string
}

func (h *LogFileRotateHook) Init() error {
	if err := h.LogFileHook.Init(); err != nil {
		return err
	}
	h.filePaths = make([]string, h.RotateNum)
	for i := 1; i < h.RotateNum; i++ {
		fileName := fmt.Sprintf("%s.%d", h.FileName, i)
		filePath := filepath.Join(h.FileDir, fileName)
		h.filePaths[i] = filePath
	}
	h.filePaths[0] = filepath.Join(h.FileDir, h.FileName)
	return nil
}

func (h *LogFileRotateHook) rotate() {
	for i := h.RotateNum - 1; i > 0; i-- {
		filePath0 := h.filePaths[i-1]
		if _, err := os.Lstat(filePath0); err != nil {
			continue
		}
		filePath1 := h.filePaths[i]
		os.Rename(filePath0, filePath1)
	}
	h.LogFileHook.DeInit()
	h.LogFileHook.Init()
}

func (h *LogFileRotateHook) Fire(e *logrus.Entry) error {
	if err := h.LogFileHook.Fire(e); err != nil {
		return err
	}
	if h.LogFileHook.Written() >= h.RotateSize {
		h.rotate()
	}
	return nil
}
