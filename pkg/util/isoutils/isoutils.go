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
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/kdomanski/iso9660"
	"github.com/mogaika/udf"
	"gopkg.in/ini.v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/imagetools"
)

// ========== 2. 新增结构化返回结果（包含发行版、版本号、架构） ==========
type ISOInfo struct {
	Distro   string // 发行版（如 CentOS、Ubuntu Server）
	Version  string // 版本号（如 7.9、22.04 LTS、2022）
	Arch     string // 架构（如 x86_64、riscv64、arm64）
	Language string // 语言（如 en-US、zh-CN）
}

// ISO格式类型
type ISOFormat string

const (
	ISOFormatUnknown ISOFormat = "unknown"
	ISOFormatUDF     ISOFormat = "udf"
	ISOFormatISO9660 ISOFormat = "iso9660"
)

// ========== 3. 优化ISOFileReader：增加缓存、日志、架构识别 ==========
type ISOFileReader struct {
	format     ISOFormat
	img        *udf.Udf
	iso9660Img *iso9660.Image // ISO9660格式的读取器
	reader     io.Reader
	cache      sync.Map // 缓存已读取的文件内容：key=文件路径，value=文件内容
}

// isIsoFile 检测ISO格式（UDF或ISO9660）
func isIsoFile(readerAt io.ReaderAt) (bool, error) {
	// 读取0x8000地址的内容（ISO9660的Primary Volume Descriptor位置）
	buf := make([]byte, 6)
	n, err := readerAt.ReadAt(buf, 0x8000)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("读取ISO格式标识失败: %v", err)
	}

	if n < 6 {
		return false, fmt.Errorf("读取数据不足")
	}

	// ISO9660格式：偏移0x8001-0x8005应该是"CD001"
	if bytes.Equal(buf[1:6], []byte("CD001")) {
		return true, nil
	}
	return false, nil
}

// NewISOFileReader 初始化ISO读取器（新增格式检测和日志配置）
func NewISOFileReader(reader io.Reader) (*ISOFileReader, error) {
	readerAt, ok := reader.(io.ReaderAt)
	if !ok {
		return nil, fmt.Errorf("reader is not io.ReaderAt")
	}

	// 检测ISO格式
	isIso, err := isIsoFile(readerAt)
	if err != nil {
		return nil, err
	}

	if !isIso {
		return nil, fmt.Errorf("ISO镜像格式不正确")
	}

	ret := &ISOFileReader{
		format: ISOFormatISO9660,
		reader: reader,
		cache:  sync.Map{},
	}

	if isUdfFile(readerAt) {
		ret.format = ISOFormatUDF
		ret.img = udf.NewUdfFromReader(readerAt)
		return ret, nil
	}

	isoImg, err := iso9660.OpenImage(readerAt)
	if err != nil {
		return nil, fmt.Errorf("打开ISO9660镜像失败: %v", err)
	}
	ret.iso9660Img = isoImg

	return ret, nil
}

// ISO9660FileInfo ISO9660文件信息
type ISO9660FileInfo struct {
	Name     string // 文件名
	IsDir    bool   // 是否为目录
	Size     int64  // 文件大小（字节）
	Location int64  // 文件在ISO中的位置（字节偏移，使用库时可能为0）
}

func (r *ISOFileReader) list(path string) ([]ISO9660FileInfo, error) {
	if r.format == ISOFormatISO9660 {
		return r.listISO9660Dir(path)
	}
	return r.listUdfDir(path)
}

// FileExists 检查ISO内指定路径的文件是否存在（支持UDF和ISO9660）
func (r *ISOFileReader) FileExists(path string) bool {
	if r.format == ISOFormatISO9660 {
		_, err := r.findISO9660File(path)
		return err == nil
	}
	_, err := r.GetFile(path)
	return err == nil
}

// ReadFileContent 读取ISO内指定文件的内容（新增缓存、日志，支持UDF和ISO9660）
func (r *ISOFileReader) ReadFileContent(path string) (string, error) {
	// 优先从缓存读取
	if cacheVal, ok := r.cache.Load(path); ok {
		log.Debugf("从缓存读取文件内容: %s", path)
		return cacheVal.(string), nil
	}

	var content string
	var err error

	// 根据格式选择相应的读取方法
	if r.format == ISOFormatISO9660 {
		content, err = r.readISO9660FileContent(path)
	} else if r.format == ISOFormatUDF {
		content, err = r.readUdfFileContent(path)
	} else {
		return "", fmt.Errorf("未知的ISO格式: %s", r.format)
	}

	if err != nil {
		return "", err
	}

	// 写入缓存
	r.cache.Store(path, content)
	log.Debugf("读取文件%s内容（长度: %d）并缓存", path, len(content))

	return content, nil
}

// ========== 6. 核心识别函数：整合版本号、架构、日志、缓存 ==========
func DetectOSFromISO(r io.Reader) (*ISOInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("DetectOSFromISO panic error: %v", r)
		}
	}()

	result := &ISOInfo{}

	// 初始化读取器
	reader, err := NewISOFileReader(r)
	if err != nil {
		return nil, err
	}

	// ========== 识别Windows系列 ==========
	if reader.FileExists("sources/install.wim") {
		return DetectWindowsEdition(reader)
	}

	files, err := reader.list("/")
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fileName := file.Name

		if fileName == ".treeinfo" {
			content, _ := reader.ReadFileContent(fileName)
			result = getOsInfoByIniFile(content)
			break
		}
	}

	if reader.FileExists(".disk/info") {
		content, _ := reader.ReadFileContent(".disk/info")
		info := imagetools.NormalizeImageInfo(content, "", "", "", "")
		result = &ISOInfo{
			Distro:   info.OsDistro,
			Version:  info.OsVersion,
			Arch:     info.OsArch,
			Language: info.OsLang,
		}
	}

	if len(result.Distro) == 0 || result.Distro == imagetools.OS_DIST_OTHER_LINUX {
		realeaseFile := ""
		if reader.FileExists("dists") {
			files, err := reader.list("dists")
			if err != nil {
				return nil, err
			}

			for _, file := range files {
				log.Debugf("file: %s", file.Name)
				if !file.IsDir {
					continue
				}
				subFiles, err := reader.list(fmt.Sprintf("dists/%s", file.Name))
				if err != nil {
					return nil, err
				}
				for _, subFile := range subFiles {
					if subFile.Name == "Release" {
						realeaseFile = fmt.Sprintf("dists/%s/%s", file.Name, subFile.Name)
						break
					}
				}
				if realeaseFile != "" {
					break
				}
			}
		}
		if realeaseFile != "" {
			content, _ := reader.ReadFileContent(realeaseFile)
			result = getOsInfoByReleaseFile(content)
		} else if reader.FileExists("boot/grub2/grub.cfg") {
			content, _ := reader.ReadFileContent("boot/grub2/grub.cfg")
			result = getOsInfoByGrub(content)
		} else if reader.FileExists("EFI/BOOT/grub.cfg") {
			content, _ := reader.ReadFileContent("EFI/BOOT/grub.cfg")
			result = getOsInfoByGrub(content)
		} else if reader.FileExists("isolinux/isolinux.cfg") {
			content, _ := reader.ReadFileContent("isolinux/isolinux.cfg")
			result = getOsInfoByIsoLinux(content)
		}
	}

	return result, nil
}

func getOsInfoByReleaseFile(content string) *ISOInfo {
	result := &ISOInfo{}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Origin:") {
			result.Distro = strings.TrimSpace(strings.TrimPrefix(line, "Origin:"))
		}
		if strings.HasPrefix(line, "Label:") && len(result.Distro) == 0 {
			result.Distro = strings.TrimSpace(strings.TrimPrefix(line, "Label:"))
		}
		if strings.HasPrefix(line, "Version:") {
			result.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
		if strings.HasPrefix(line, "Architectures:") {
			result.Arch = strings.TrimSpace(strings.TrimPrefix(line, "Architectures:"))
			result.Arch = detectArchitecture(strings.ToLower(result.Arch), result.Arch)
		}
	}
	return result
}

func getOsInfoByIniFile(content string) *ISOInfo {
	cfg, err := ini.Load([]byte(content))
	if err != nil {
		// 兼容手动解析（应对部分非标准 INI 格式的 .treeinfo）
		return parseTreeInfoFallback(content)
	}
	release := cfg.Section("release")
	ret := &ISOInfo{}
	ret.Distro = release.Key("name").String()
	ret.Version = release.Key("version").String()
	general := cfg.Section("general")
	ret.Arch = general.Key("arch").String()
	if len(ret.Version) == 0 {
		ret.Version = general.Key("version").String()
	}
	return ret
}

func parseTreeInfoFallback(content string) *ISOInfo {
	result := &ISOInfo{}
	info := strings.Split(content, "\n")
	for _, line := range info {
		if strings.HasPrefix(line, "arch =") {
			result.Arch = strings.TrimPrefix(line, "arch = ")
		}
		if strings.HasPrefix(line, "version =") {
			result.Version = strings.TrimPrefix(line, "version = ")
		}
		if strings.HasPrefix(line, "name =") {
			result.Distro = strings.TrimPrefix(line, "name = ")
		}
	}
	return result
}

func getOsInfoByIsoLinux(content string) *ISOInfo {
	result := &ISOInfo{}
	lowerContent := strings.ToLower(content)
	// 5.1 识别发行版（关键词匹配）
	result.Distro = detectDistro(lowerContent)

	// 5.2 识别版本号（正则提取）
	result.Version = detectVersion(content)

	// 5.3 识别 CPU 架构（关键词+正则）
	result.Arch = detectArchitecture(lowerContent, content)
	return result
}

func getOsInfoByGrub(content string) *ISOInfo {
	result := &ISOInfo{}
	lowerContent := strings.ToLower(content)
	result.Distro = detectDistro(lowerContent)
	result.Version = detectGrubVersion(content)
	result.Arch = detectArchitecture(lowerContent, content)
	return result
}

func detectDistro(lowerContent string) string {
	info := imagetools.NormalizeImageInfo(lowerContent, "", "", "", "")
	return info.OsDistro
}

// detectVersion 从配置内容中提取版本号
func detectVersion(content string) string {
	// 匹配版本号的正则（支持 x x.y、x.y.z、x.y-LTS 等格式）
	versionRegex := regexp.MustCompile(`(\d+(\.\d+(\.\d+)?)?(-[A-Za-z0-9]+)?)`)

	// 优先从启动标题（label/menu label）中提取
	labelLines := regexp.MustCompile(`(?i)menu label .+Install.+`).FindAllStringSubmatch(content, -1)
	for _, line := range labelLines {
		log.Debugf("line: %s", line)
		if len(line) >= 1 {
			version := versionRegex.FindString(line[0])
			if version != "" {
				return version
			}
		}
	}

	// 从整个内容中提取第一个匹配的版本号
	return versionRegex.FindString(content)
}

// detectArchitecture 识别 CPU 架构
func detectArchitecture(lowerContent, rawContent string) string {
	// 架构关键词映射
	archKeywords := map[string][]string{
		"x86_64":  {"x86_64", "amd64"},
		"aarch64": {"aarch64", "arm64"},
		"i386":    {"i386", "i686"},
		"armhfp":  {"armhfp", "armv7"},
		"ppc64le": {"ppc64le"},
		"s390x":   {"s390x"},
	}

	for arch, keywords := range archKeywords {
		for _, kw := range keywords {
			if strings.Contains(lowerContent, kw) {
				return arch
			}
		}
	}

	// 从内核文件名（vmlinuz/initrd）中提取
	kernelRegex := regexp.MustCompile(`vmlinuz-([a-zA-Z0-9_]+)`)
	match := kernelRegex.FindStringSubmatch(rawContent)
	if len(match) >= 2 {
		return match[1]
	}

	return ""
}

// detectGrubVersion 从 grub.cfg 提取版本号
func detectGrubVersion(content string) string {
	// 匹配版本号的正则（支持 x x.y、x.y.z、x.y-LTS、x.y.z-xxx 等格式）
	versionRegex := regexp.MustCompile(`(\d+(\.\d+(\.\d+)?)?(-[A-Za-z0-9]+)?)`)

	// 优先从 GRUB 菜单标题（menuentry）中提取（准确性更高）
	menuEntryRegex := regexp.MustCompile(`(?i)menuentry\s+["'](.+?)["']`)
	menuEntries := menuEntryRegex.FindAllStringSubmatch(content, -1)
	for _, entry := range menuEntries {
		log.Debugf("entry: %s", entry)
		if len(entry) >= 1 {
			version := versionRegex.FindString(entry[0])
			if version != "" {
				return version
			}
		}
	}

	// 从内核文件名/参数中提取
	kernelLines := regexp.MustCompile(`linux\s+.+`).FindAllString(content, -1)
	for _, line := range kernelLines {
		version := versionRegex.FindString(line)
		if version != "" {
			return version
		}
	}

	// 最后从整个内容中提取第一个匹配的版本号
	return versionRegex.FindString(content)
}
