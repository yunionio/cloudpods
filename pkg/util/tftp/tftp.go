// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tftp implements a read-only TFTP server.
package tftp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	// DefaultWriteTimeout is the duration a client has to acknowledge
	// a data packet from the server. This can be overridden by
	// setting Server.WriteTimeout.
	DefaultWriteTimeout = 2 * time.Second
	// DefaultWriteAttempts is the maximum number of times a single
	// packet will be (re)sent before timing out a client. This can be
	// overridden by setting Server.WriteAttempts.
	DefaultWriteAttempts = 5
	// DefaultBlockSize is the maximum block size used to send data to
	// clients. The server will respect a request for a smaller block
	// size, but requests for larger block sizes will be clamped to
	// DefaultBlockSize. This can be overridden by setting
	// Server.MaxBlockSize.
	DefaultBlockSize = 1450

	// maxErrorSize is the largest error message string that will be
	// sent to the client without truncation.
	maxErrorSize = 500
)

// A Handler provides bytes for a file.
//
// If size is non-zero, it must be equal to the number of bytes in
// file. The server will offer the "tsize" extension to clients that
// request it.
//
// Note that some clients (particularly firmware TFTP clients) can be
// very capricious about servers not supporting all the options that
// they request, so passing a size of 0 may cause TFTP transfers to
// fail for some clients.
type Handler func(path string, clientAddr net.Addr) (file io.ReadCloser, size int64, err error)

// A Server defines parameters for running a TFTP server.
type Server struct {
	Handler Handler // handler to invoke for requests

	// WriteTimeout sets the duration to wait for the client to
	// acknowledge a data packet. Defaults to DefaultWriteTimeout.
	WriteTimeout time.Duration
	// WriteAttempts sets how many times a packet will be (re)sent
	// before timing out the client and aborting the transfer. If 0,
	// uses DefaultWriteAttempts.
	WriteAttempts int
	// MaxBlockSize sets the maximum block size used for file
	// transfers. If 0, uses DefaultBlockSize.
	MaxBlockSize int64

	// InfoLog specifies an optional logger for informational
	// messages. If nil, informational messages are suppressed.
	InfoLog func(msg string)
	// TransferLog specifies an optional logger for completed
	// transfers. A successful transfer is logged with err == nil. If
	// nil, transfer logs are suppressed.
	TransferLog func(clientAddr net.Addr, path string, err error)

	// Dial specifies a function to use when setting up a "connected"
	// UDP socket to a TFTP client. While this is mostly here for
	// testing, it can also be used to implement advanced relay
	// functionality (e.g. serving TFTP through SOCKS). If nil,
	// net.Dial is used.
	Dial func(network, addr string) (net.Conn, error)
}

// ListenAndServe listens on the UDP network address addr and then
// calls Serve to handle TFTP requests. If addr is blank, ":69" is
// used.
func (s *Server) ListenAndServe(addr string) error {
	if addr == "" {
		addr = ":69"
	}
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer l.Close()
	s.infoLog("TFTP listening on %s", l.LocalAddr())
	return s.Serve(l)
}

// Serve accepts requests on listener l, creating a new transfer
// goroutine for each. The transfer goroutines use s.Handler to get
// bytes, and transfers them to the client.
func (s *Server) Serve(l net.PacketConn) error {
	if s.Handler == nil {
		return errors.New("can't serve, Handler is nil")
	}
	if err := l.SetDeadline(time.Time{}); err != nil {
		return err
	}
	buf := make([]byte, 512)
	for {
		n, addr, err := l.ReadFrom(buf)
		if err != nil {
			return err
		}

		req, err := parseRRQ(buf[:n])
		if err != nil {
			s.infoLog("bad request from %q: %s", addr, err)
			continue
		}

		go s.transferAndLog(addr, req)
	}

}

func (s *Server) infoLog(msg string, args ...interface{}) {
	if s.InfoLog != nil {
		s.InfoLog(fmt.Sprintf(msg, args...))
	}
}

func (s *Server) transferLog(addr net.Addr, path string, err error) {
	if s.TransferLog != nil {
		s.TransferLog(addr, path, err)
	}
}

func (s *Server) transferAndLog(addr net.Addr, req *rrq) {
	err := s.transfer(addr, req)
	if err != nil {
		err = fmt.Errorf("%q: %s", addr, err)
	}
	s.transferLog(addr, req.Filename, err)
}

func (s *Server) transfer(addr net.Addr, req *rrq) error {
	d := s.Dial
	if d == nil {
		d = net.Dial
	}
	conn, err := d("udp", addr.String())
	if err != nil {
		return fmt.Errorf("creating socket: %s", err)
	}
	defer conn.Close()

	file, size, err := s.Handler(req.Filename, addr)
	if err != nil {
		conn.Write(tftpError("failed to get file"))
		return fmt.Errorf("getting file bytes: %s", err)
	}
	defer file.Close()

	var b bytes.Buffer
	if req.BlockSize != 0 || (req.WantSize && size != 0) {
		// Client requested options, need to OACK them before sending
		// data.
		b.WriteByte(0)
		b.WriteByte(6)

		if req.BlockSize != 0 {
			maxBlockSize := s.MaxBlockSize
			if maxBlockSize <= 0 {
				maxBlockSize = DefaultBlockSize
			}
			if req.BlockSize > maxBlockSize {
				s.infoLog("clamping blocksize to %q: %d -> %d", addr, req.BlockSize, maxBlockSize)
				req.BlockSize = maxBlockSize
			}

			b.WriteString("blksize")
			b.WriteByte(0)
			b.WriteString(strconv.FormatInt(req.BlockSize, 10))
			b.WriteByte(0)
		}

		if req.WantSize && size != 0 {
			b.WriteString("tsize")
			b.WriteByte(0)
			b.WriteString(strconv.FormatInt(size, 10))
			b.WriteByte(0)
		}

		if err := s.send(conn, b.Bytes(), 0); err != nil {
			return fmt.Errorf("sending OACK: %s", err)
		}
		b.Reset()
	}
	if req.BlockSize == 0 {
		// Client didn't negotiate, use classic blocksize from RFC.
		req.BlockSize = 512
	}

	seq := uint16(1)
	b.Grow(int(req.BlockSize + 4))
	b.WriteByte(0)
	b.WriteByte(3)
	for {
		b.Truncate(2)
		if err = binary.Write(&b, binary.BigEndian, seq); err != nil {
			conn.Write(tftpError("internal server error"))
			return fmt.Errorf("writing seqnum: %s", err)
		}
		n, err := io.CopyN(&b, file, req.BlockSize)
		if err != nil && err != io.EOF {
			conn.Write(tftpError("internal server error"))
			return fmt.Errorf("reading bytes for block %d: %s", seq, err)
		}
		if err = s.send(conn, b.Bytes(), seq); err != nil {
			conn.Write(tftpError("timeout"))
			return fmt.Errorf("sending data packet %d: %s", seq, err)
		}
		seq++
		if n < req.BlockSize {
			// Transfer complete
			return nil
		}
	}
}

func (s *Server) send(conn net.Conn, b []byte, seq uint16) error {
	timeout := s.WriteTimeout
	if timeout <= 0 {
		timeout = DefaultWriteTimeout
	}
	attempts := s.WriteAttempts
	if attempts <= 0 {
		attempts = DefaultWriteAttempts
	}

Attempt:
	for attempt := 0; attempt < attempts; attempt++ {
		if _, err := conn.Write(b); err != nil {
			return err
		}

		conn.SetReadDeadline(time.Now().Add(timeout))

		var recv [256]byte
		for {
			n, err := conn.Read(recv[:])
			if err != nil {
				if t, ok := err.(net.Error); ok && t.Timeout() {
					continue Attempt
				}
				return err
			}

			if n < 4 { // packet too small
				continue
			}
			switch binary.BigEndian.Uint16(recv[:2]) {
			case 4:
				if binary.BigEndian.Uint16(recv[2:4]) == seq {
					return nil
				}
			case 5:
				msg, _, _ := tftpStr(recv[4:])
				return fmt.Errorf("client aborted transfer: %s", msg)
			}
		}
	}

	return errors.New("timeout waiting for ACK")
}

type rrq struct {
	Filename  string
	BlockSize int64
	WantSize  bool
}

func parseRRQ(bs []byte) (*rrq, error) {
	// Smallest a useful TFTP packet can be is 6 bytes: 2b opcode, 1b
	// filename, 1b null, 1b mode, 1b null.
	if len(bs) < 6 || binary.BigEndian.Uint16(bs[:2]) != 1 {
		return nil, errors.New("not an RRQ packet")
	}

	fname, bs, err := tftpStr(bs[2:])
	if err != nil {
		return nil, fmt.Errorf("reading filename: %s", err)
	}

	mode, bs, err := tftpStr(bs)
	if err != nil {
		return nil, fmt.Errorf("reading mode: %s", err)
	}
	if mode != "octet" {
		// Only support octet mode, because in practice that's the
		// only remaining sensible use of TFTP (i.e. PXE booting)
		return nil, fmt.Errorf("unsupported transfer mode %q", mode)
	}

	req := &rrq{
		Filename: fname,
	}

	for len(bs) > 0 {
		opt, rest, err := tftpStr(bs)
		if err != nil {
			return nil, fmt.Errorf("reading option name: %s", err)
		}
		bs = rest
		val, rest, err := tftpStr(bs)
		if err != nil {
			return nil, fmt.Errorf("reading option %q value: %s", opt, err)
		}
		bs = rest
		if opt != "blksize" {
			if opt == "tsize" {
				req.WantSize = true
			}
			continue
		}
		size, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("non-integer block size value %q", val)
		}
		if size < 8 || size > 65464 {
			return nil, fmt.Errorf("unsupported block size %q", size)
		}
		req.BlockSize = size
	}

	return req, nil
}

// tftpError constructs an ERROR packet.
//
// The error is coerced to the sensible subset of "netascii", namely
// the printable ASCII characters plus newline.
func tftpError(msg string) []byte {
	if len(msg) > maxErrorSize {
		msg = msg[:maxErrorSize]
	}
	var ret bytes.Buffer
	ret.Grow(len(msg) + 5)
	ret.Write([]byte{0, 5, 0, 0}) // generic "see message" error packet
	for _, b := range msg {
		switch {
		case b >= 0x20 && b <= 0x7E:
			ret.WriteRune(b)
		case b == '\r':
			// Assume this is the start of a CRLF sequence and just
			// swallow the CR. The LF will output CRLF, see
			// below. Also, please stop using CRLF line termination in
			// Go.
		case b == '\n':
			ret.WriteString("\r\n")
		default:
			ret.WriteByte('?')
		}
	}
	ret.WriteByte(0)
	return ret.Bytes()
}

// tftpStr extracts a null-terminated string from the given bytes, and
// returns any remaining bytes.
//
// String content is checked to be a "read-useful" subset of
// "netascii", itself a subset of ASCII. Specifically, all byte values
// must fall in the range 0x20 to 0x7E inclusive.
func tftpStr(bs []byte) (str string, remaining []byte, err error) {
	for i, b := range bs {
		if b == 0 {
			return string(bs[:i]), bs[i+1:], nil
		} else if b < 0x20 || b > 0x7E {
			return "", nil, fmt.Errorf("invalid netascii byte %q at offset %d", b, i)
		}
	}
	return "", nil, errors.New("no null terminated string found")
}
