/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package util

import (
	"go/build"
	"os"
	"runtime"
	"strings"
)

const (
	PathSeparator     = string(os.PathSeparator)
	PathListSeparator = string(os.PathListSeparator)
)

var (
	goRoot = build.Default.GOROOT
	goPath = build.Default.GOPATH //获取实际编译时的GOPATH值
)

type fileUtil struct {
}

var FileUtil = &fileUtil{}

func (fileUtil *fileUtil) Exists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func (fileUtil *fileUtil) Search(relativePath string) (path string) {
	if strings.Contains(runtime.GOOS, "windows") {
		relativePath = strings.ReplaceAll(relativePath, "/", "\\")
	}

	if fileUtil.Exists(goPath) {
		for _, s := range strings.Split(goPath, PathListSeparator) {
			path = s + PathSeparator + "src" + PathSeparator + relativePath
			if fileUtil.Exists(path) {
				return path
			}
		}
	}

	if fileUtil.Exists(goPath) {
		for _, s := range strings.Split(goPath, PathListSeparator) {
			path = s + PathSeparator + "pkg" + PathSeparator + relativePath
			if fileUtil.Exists(path) {
				return path
			}
		}
	}

	//if workDir, _ := os.Getwd(); fileUtil.Exists(workDir) {
	//	path = workDir + PathSeparator + "src" + PathSeparator + relativePath
	//	if fileUtil.Exists(path) {
	//		return path
	//	}
	//}

	//if fileUtil.Exists(goRoot) {
	//	path = goRoot + PathSeparator + "src" + PathSeparator + relativePath
	//	if fileUtil.Exists(path) {
	//		return path
	//	}
	//}

	return ""
}
