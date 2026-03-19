package framestream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
)

const CONTROL_ACCEPT = 0x01
const CONTROL_START = 0x02
const CONTROL_STOP = 0x03
const CONTROL_READY = 0x04
const CONTROL_FINISH = 0x05

const CONTROL_FIELD_CONTENT_TYPE = 0x01

const CONTROL_FRAME_LENGTH_MAX = 512

type ControlFrame struct {
	ControlType  uint32
	ContentTypes [][]byte
}

var ControlStart = ControlFrame{ControlType: CONTROL_START}
var ControlStop = ControlFrame{ControlType: CONTROL_STOP}
var ControlReady = ControlFrame{ControlType: CONTROL_READY}
var ControlAccept = ControlFrame{ControlType: CONTROL_ACCEPT}
var ControlFinish = ControlFrame{ControlType: CONTROL_FINISH}

func (c *ControlFrame) Encode(w io.Writer) (err error) {
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.BigEndian, c.ControlType)
	if err != nil {
		return
	}
	for _, ctype := range c.ContentTypes {
		err = binary.Write(&buf, binary.BigEndian, uint32(CONTROL_FIELD_CONTENT_TYPE))
		if err != nil {
			return
		}

		err = binary.Write(&buf, binary.BigEndian, uint32(len(ctype)))
		if err != nil {
			return
		}

		_, err = buf.Write(ctype)
		if err != nil {
			return
		}
	}

	err = binary.Write(w, binary.BigEndian, uint32(0))
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, uint32(buf.Len()))
	if err != nil {
		return
	}
	_, err = buf.WriteTo(w)
	return
}

func (c *ControlFrame) EncodeFlush(w *bufio.Writer) error {
	if err := c.Encode(w); err != nil {
		return err
	}
	return w.Flush()
}

func (c *ControlFrame) Decode(r io.Reader) (err error) {
	var cflen uint32
	err = binary.Read(r, binary.BigEndian, &cflen)
	if err != nil {
		return
	}

	if cflen > CONTROL_FRAME_LENGTH_MAX {
		return ErrDecode
	}

	if cflen < 4 {
		return ErrDecode
	}

	err = binary.Read(r, binary.BigEndian, &c.ControlType)
	if err != nil {
		return
	}

	cflen -= 4
	if cflen > 0 {
		cfields := make([]byte, int(cflen))
		_, err = io.ReadFull(r, cfields)
		if err != nil {
			return
		}

		for len(cfields) > 8 {
			cftype := binary.BigEndian.Uint32(cfields[:4])
			cfields = cfields[4:]
			if cftype != CONTROL_FIELD_CONTENT_TYPE {
				return ErrDecode
			}

			cflen := int(binary.BigEndian.Uint32(cfields[:4]))
			cfields = cfields[4:]
			if cflen > len(cfields) {
				return ErrDecode
			}

			c.ContentTypes = append(c.ContentTypes, cfields[:cflen])
			cfields = cfields[cflen:]
		}

		if len(cfields) > 0 {
			return ErrDecode
		}
	}
	return
}

func (c *ControlFrame) DecodeEscape(r io.Reader) error {
	var zero uint32
	err := binary.Read(r, binary.BigEndian, &zero)
	if err != nil {
		return err
	}
	if zero != 0 {
		return ErrDecode
	}
	return c.Decode(r)
}

func (c *ControlFrame) DecodeTypeEscape(r io.Reader, ctype uint32) error {
	err := c.DecodeEscape(r)
	if err != nil {
		return err
	}

	if ctype != c.ControlType {
		return ErrDecode
	}

	return nil
}

// ChooseContentType selects a content type from the ControlFrame which
// also exists in the supplied ctypes. Preference is given to values occurring
// earliest in ctypes.
//
// ChooseContentType returns the chosen content type, which may be nil, and
// a bool value indicating whether a matching type was found.
//
// If either the ControlFrame types or ctypes is empty, ChooseContentType
// returns nil as a matching content type.
func (c *ControlFrame) ChooseContentType(ctypes [][]byte) (typ []byte, found bool) {
	if c.ContentTypes == nil || ctypes == nil {
		return nil, true
	}
	tm := make(map[string]bool)
	for _, cfctype := range c.ContentTypes {
		tm[string(cfctype)] = true
	}
	for _, ctype := range ctypes {
		if tm[string(ctype)] {
			return ctype, true
		}
	}
	return nil, false
}

func (c *ControlFrame) MatchContentType(ctype []byte) bool {
	for _, cfctype := range c.ContentTypes {
		if bytes.Compare(ctype, cfctype) == 0 {
			return true
		}
	}
	return len(c.ContentTypes) == 0
}

func (c *ControlFrame) SetContentTypes(ctypes [][]byte) {
	c.ContentTypes = ctypes
}

func (c *ControlFrame) SetContentType(ctype []byte) {
	if ctype != nil {
		c.SetContentTypes([][]byte{ctype})
	} else {
		c.ContentTypes = nil
	}
}
