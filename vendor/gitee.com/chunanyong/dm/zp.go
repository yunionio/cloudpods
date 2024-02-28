/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gitee.com/chunanyong/dm/util"
)

const (
	MAX_FILE_SIZE = 100 * 1024 * 1024
	FLUSH_SIZE    = 32 * 1024
)

type goRun interface {
	doRun()
}

type logWriter struct {
	flushQueue chan []byte
	date       string
	logFile    *os.File
	flushFreq  int
	filePath   string
	filePrefix string
	buffer     *Dm_build_931
}

func (lw *logWriter) doRun() {
	defer func() {
		lw.beforeExit()
		lw.closeCurrentFile()
	}()

	i := 0
	for {
		var ibytes []byte

		select {
		case ibytes = <-lw.flushQueue:
			if LogLevel != LOG_OFF {
				if i == LogFlushQueueSize {
					lw.doFlush(lw.buffer)
					i = 0
				} else {
					lw.buffer.Dm_build_957(ibytes, 0, len(ibytes))
					i++
				}
			}
		case <-time.After(time.Duration(LogFlushFreq) * time.Millisecond):
			if LogLevel != LOG_OFF && lw.buffer.Dm_build_936() > 0 {
				lw.doFlush(lw.buffer)
				i = 0
			}

		}

	}
}

func (lw *logWriter) doFlush(buffer *Dm_build_931) {
	if lw.needCreateNewFile() {
		lw.closeCurrentFile()
		lw.logFile = lw.createNewFile()
	}
	if lw.logFile != nil {
		buffer.Dm_build_951(lw.logFile, buffer.Dm_build_936())
	}
}
func (lw *logWriter) closeCurrentFile() {
	if lw.logFile != nil {
		lw.logFile.Close()
		lw.logFile = nil
	}
}
func (lw *logWriter) createNewFile() *os.File {
	lw.date = time.Now().Format("2006-01-02")
	fileName := lw.filePrefix + "_" + lw.date + "_" + strconv.Itoa(time.Now().Nanosecond()) + ".log"
	lw.filePath = LogDir
	if len(lw.filePath) > 0 {
		if _, err := os.Stat(lw.filePath); err != nil {
			os.MkdirAll(lw.filePath, 0755)
		}
		if _, err := os.Stat(lw.filePath + fileName); err != nil {
			logFile, err := os.Create(lw.filePath + fileName)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			return logFile
		}
	}
	return nil
}
func (lw *logWriter) needCreateNewFile() bool {
	now := time.Now().Format("2006-01-02")
	fileInfo, err := lw.logFile.Stat()
	return now != lw.date || err != nil || lw.logFile == nil || fileInfo.Size() > int64(MAX_FILE_SIZE)
}
func (lw *logWriter) beforeExit() {
	close(lw.flushQueue)
	var ibytes []byte
	for ibytes = <-lw.flushQueue; ibytes != nil; ibytes = <-lw.flushQueue {
		lw.buffer.Dm_build_957(ibytes, 0, len(ibytes))
		if lw.buffer.Dm_build_936() >= LogBufferSize {
			lw.doFlush(lw.buffer)
		}
	}
	if lw.buffer.Dm_build_936() > 0 {
		lw.doFlush(lw.buffer)
	}
}

func (lw *logWriter) WriteLine(msg string) {
	var b = []byte(strings.TrimSpace(msg) + util.LINE_SEPARATOR)
	lw.flushQueue <- b
}
