/*
 * Copyright (c) 2014,2019 by Farsight Security, Inc.
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

//go:generate ./genproto.sh

package dnstap

const outputChannelSize = 32

// FSContentType is the FrameStream content type for dnstap protobuf data.
var FSContentType = []byte("protobuf:dnstap.Dnstap")

// An Input is a source of dnstap data. It provides validation of the
// content type and will present any data read or received on the channel
// provided to the ReadInto method.
type Input interface {
	ReadInto(chan []byte)
	Wait()
}

// An Output is a destination for dnstap data. It accepts data on the channel
// returned from the GetOutputChannel method. The RunOutputLoop() method
// processes data received on this channel, and returns after the Close()
// method is called.
type Output interface {
	GetOutputChannel() chan []byte
	RunOutputLoop()
	Close()
}

// A Logger prints a formatted log message to the destination of the
// implementation's choice. A Logger may be provided for some Input and
// Output implementations for visibility into their ReadInto() and
// RunOutputLoop() loops.
//
// The result of log.New() satisfies the Logger interface.
type Logger interface {
	Printf(format string, v ...interface{})
}

type nullLogger struct{}

func (n nullLogger) Printf(format string, v ...interface{}) {}
