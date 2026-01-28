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

package isoutils

import (
	"fmt"
	"io"
	"strings"

	"github.com/mogaika/udf"

	"yunion.io/x/log"
)

// isUdfFile 检测是否为UDF格式
func isUdfFile(readerAt io.ReaderAt) bool {
	defer func() {
		recover()
	}()

	img := udf.NewUdfFromReader(readerAt)
	files := img.ReadDir(nil)
	if len(files) == 0 {
		return false
	}
	return true
}

// findUdfDir 在UDF中查找目录，返回目录的 FileEntry
func (r *ISOFileReader) findUdfDir(path string) ([]udf.File, error) {
	if r.format != ISOFormatUDF || r.img == nil {
		return nil, fmt.Errorf("UDF格式未初始化")
	}

	// 规范化路径
	path = strings.Trim(path, "/")
	if path == "" {
		// 根目录
		return r.img.ReadDir(nil), nil
	}

	// 查找目录路径
	parts := strings.Split(path, "/")
	var entry *udf.FileEntry = nil
	currentDirEntry := r.img.ReadDir(entry) // 从根目录开始

	for i, part := range parts {
		if part == "" {
			continue
		}

		var found *udf.File
		// 在当前目录中查找
		for idx := range currentDirEntry {
			child := &currentDirEntry[idx]
			childName := child.Name()
			// 匹配文件名（不区分大小写）
			if strings.EqualFold(childName, part) {
				found = child
				break
			}
		}

		if found == nil {
			return nil, fmt.Errorf("目录不存在: %s", part)
		}

		// 如果是最后一个部分，返回该目录的内容
		if i == len(parts)-1 {
			return found.ReadDir(), nil
		}

		// 继续查找下一级目录
		currentDirEntry = found.ReadDir()
	}

	return nil, fmt.Errorf("未找到目录: %s", path)
}

// listUdfDir 列出UDF格式指定目录下的所有文件和子目录
func (r *ISOFileReader) listUdfDir(path string) ([]ISO9660FileInfo, error) {
	if r.format != ISOFormatUDF {
		return nil, fmt.Errorf("此方法仅支持UDF格式")
	}

	// 获取目录内容
	children, err := r.findUdfDir(path)
	if err != nil {
		return nil, err
	}

	var files []ISO9660FileInfo
	for _, child := range children {
		fileInfo := ISO9660FileInfo{
			Name:     child.Name(),
			IsDir:    child.IsDir(),
			Size:     child.Size(),
			Location: 0, // 使用库时不需要直接访问位置
		}
		files = append(files, fileInfo)
	}

	return files, nil
}

// GetFile 在UDF中查找指定路径的文件
func (r *ISOFileReader) GetFile(path string) (*udf.File, error) {
	if r.format != ISOFormatUDF {
		return nil, fmt.Errorf("此方法仅支持UDF格式")
	}

	// UDF格式的原有逻辑
	parts := strings.Split(strings.Trim(path, "/"), "/")

	var entry *udf.FileEntry = nil
	currentDirEntry := r.img.ReadDir(entry) // 从根目录开始

	for i, part := range parts {
		if part == "" {
			continue
		}

		var found *udf.File
		// 在当前目录中查找
		for idx := range currentDirEntry {
			child := &currentDirEntry[idx]
			childName := child.Name()
			// 匹配文件名（UDF 文件名通常不包含版本号后缀）
			if childName == part {
				found = child
				break
			}
		}

		if found == nil {
			return nil, fmt.Errorf("文件或目录不存在: %s", part)
		}

		// 如果是最后一个部分，返回文件
		if i == len(parts)-1 {
			return found, nil
		}

		currentDirEntry = found.ReadDir()
	}

	return nil, fmt.Errorf("未找到文件: %s", path)
}

// readUdfFileContent 读取UDF格式文件内容
func (r *ISOFileReader) readUdfFileContent(path string) (string, error) {
	file, err := r.GetFile(path)
	if err != nil {
		return "", fmt.Errorf("文件%s不存在: %v", path, err)
	}

	if file.IsDir() {
		return "", fmt.Errorf("路径%s是目录，不是文件", path)
	}

	reader := file.NewReader()
	if reader == nil {
		return "", fmt.Errorf("无法读取文件%s", path)
	}

	// 读取前10KB内容（足够识别特征）
	buf := make([]byte, 10*1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		log.Errorf("读取UDF文件%s失败: %v", path, err)
		return "", fmt.Errorf("读取文件%s失败: %v", path, err)
	}

	return strings.TrimSpace(string(buf[:n])), nil
}
