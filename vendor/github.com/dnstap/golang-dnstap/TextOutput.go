/*
 * Copyright (c) 2014 by Farsight Security, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dnstap

import (
	"bufio"
	"io"
	"os"

	"google.golang.org/protobuf/proto"
)

// A TextFormatFunc renders a dnstap message into a human readable format.
type TextFormatFunc func(*Dnstap) ([]byte, bool)

// TextOutput implements a dnstap Output rendering dnstap data as text.
type TextOutput struct {
	format        TextFormatFunc
	outputChannel chan []byte
	wait          chan bool
	writer        *bufio.Writer
	log           Logger
}

// NewTextOutput creates a TextOutput writing dnstap data to the given io.Writer
// in the text format given by the TextFormatFunc format.
func NewTextOutput(writer io.Writer, format TextFormatFunc) (o *TextOutput) {
	o = new(TextOutput)
	o.format = format
	o.outputChannel = make(chan []byte, outputChannelSize)
	o.writer = bufio.NewWriter(writer)
	o.wait = make(chan bool)
	return
}

// NewTextOutputFromFilename creates a TextOutput writing dnstap data to a
// file with the given filename in the format given by format. If doAppend
// is false, the file is truncated if it already exists, otherwise the file
// is opened for appending.
func NewTextOutputFromFilename(fname string, format TextFormatFunc, doAppend bool) (o *TextOutput, err error) {
	if fname == "" || fname == "-" {
		return NewTextOutput(os.Stdout, format), nil
	}
	var writer io.Writer
	if doAppend {
		writer, err = os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	} else {
		writer, err = os.Create(fname)
	}
	if err != nil {
		return
	}
	return NewTextOutput(writer, format), nil
}

// SetLogger configures a logger for error events in the TextOutput
func (o *TextOutput) SetLogger(logger Logger) {
	o.log = logger
}

// GetOutputChannel returns the channel on which the TextOutput accepts dnstap data.
//
// GetOutputChannel satisfies the dnstap Output interface.
func (o *TextOutput) GetOutputChannel() chan []byte {
	return o.outputChannel
}

// RunOutputLoop receives dnstap data sent on the output channel, formats it
// with the configured TextFormatFunc, and writes it to the file or io.Writer
// of the TextOutput.
//
// RunOutputLoop satisfies the dnstap Output interface.
func (o *TextOutput) RunOutputLoop() {
	dt := &Dnstap{}
	for frame := range o.outputChannel {
		if err := proto.Unmarshal(frame, dt); err != nil {
			o.log.Printf("dnstap.TextOutput: proto.Unmarshal() failed: %s, returning", err)
			break
		}
		buf, ok := o.format(dt)
		if !ok {
			o.log.Printf("dnstap.TextOutput: text format function failed, returning")
			break
		}
		if _, err := o.writer.Write(buf); err != nil {
			o.log.Printf("dnstap.TextOutput: write error: %v, returning", err)
			break
		}
		o.writer.Flush()
	}
	close(o.wait)
}

// Close closes the output channel and returns when all pending data has been
// written.
//
// Close satisfies the dnstap Output interface.
func (o *TextOutput) Close() {
	close(o.outputChannel)
	<-o.wait
	o.writer.Flush()
}
