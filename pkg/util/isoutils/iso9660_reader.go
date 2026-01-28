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

	"github.com/kdomanski/iso9660"

	"yunion.io/x/log"
)

// findISO9660File 在ISO9660中查找文件，返回 *iso9660.File
func (r *ISOFileReader) findISO9660File(path string) (*iso9660.File, error) {
	if r.format != ISOFormatISO9660 || r.iso9660Img == nil {
		return nil, fmt.Errorf("ISO9660格式未初始化")
	}

	rootDir, err := r.iso9660Img.RootDir()
	if err != nil {
		return nil, fmt.Errorf("获取根目录失败: %v", err)
	}

	// 规范化路径
	path = strings.Trim(path, "/")
	if path == "" {
		return rootDir, nil
	}

	parts := strings.Split(path, "/")
	currentDir := rootDir

	for i, part := range parts {
		if part == "" {
			continue
		}

		// 获取当前目录的子项
		children, err := currentDir.GetChildren()
		if err != nil {
			return nil, fmt.Errorf("读取目录失败: %v", err)
		}

		// 查找匹配的文件或目录（不区分大小写）
		var found *iso9660.File
		for _, child := range children {
			if strings.EqualFold(child.Name(), part) {
				found = child
				break
			}
		}

		if found == nil {
			return nil, fmt.Errorf("文件或目录不存在: %s", part)
		}

		// 如果是最后一个部分，返回找到的文件
		if i == len(parts)-1 {
			return found, nil
		}

		// 检查是否为目录
		if !found.IsDir() {
			return nil, fmt.Errorf("路径中的%s不是目录", part)
		}

		currentDir = found
	}

	return currentDir, nil
}

// listISO9660Dir 列出ISO9660格式指定目录下的所有文件和子目录
func (r *ISOFileReader) listISO9660Dir(path string) ([]ISO9660FileInfo, error) {
	if r.format != ISOFormatISO9660 {
		return nil, fmt.Errorf("此方法仅支持ISO9660格式")
	}

	// 获取目录
	dir, err := r.findISO9660File(path)
	if err != nil {
		return nil, err
	}

	if !dir.IsDir() {
		return nil, fmt.Errorf("路径%s不是目录", path)
	}

	// 获取子项
	children, err := dir.GetChildren()
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
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

// readISO9660FileContent 读取ISO9660格式文件内容
func (r *ISOFileReader) readISO9660FileContent(path string) (string, error) {
	file, err := r.findISO9660File(path)
	if err != nil {
		return "", fmt.Errorf("文件%s不存在: %v", path, err)
	}

	if file.IsDir() {
		return "", fmt.Errorf("路径%s是目录，不是文件", path)
	}

	reader := file.Reader()
	if reader == nil {
		return "", fmt.Errorf("无法读取文件%s", path)
	}

	// 读取前10KB内容（足够识别特征）
	buf := make([]byte, 10*1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		log.Errorf("读取ISO9660文件%s失败: %v", path, err)
		return "", fmt.Errorf("读取文件%s失败: %v", path, err)
	}

	return strings.TrimSpace(string(buf[:n])), nil
}
