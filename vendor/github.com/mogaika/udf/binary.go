package udf

import (
	"encoding/binary"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func r_u8(b []byte) uint8 {
	return b[0]
}

func r_i8(b []byte) int8 {
	return int8(r_u8(b))
}

var rl_u64 = binary.LittleEndian.Uint64

func rl_u48(b []byte) uint64 {
	var buf [8]byte
	copy(buf[:], b[:6])
	return rl_u64(buf[:])
}

var rl_u32 = binary.LittleEndian.Uint32
var rl_u16 = binary.LittleEndian.Uint16

func rl_i64(b []byte) int64 {
	return int64(rl_u64(b))
}

func rl_i32(b []byte) int32 {
	return int32(rl_u32(b))
}

func rl_i16(b []byte) int16 {
	return int16(rl_u16(b))
}

var rb_u64 = binary.BigEndian.Uint64
var rb_u32 = binary.BigEndian.Uint32
var rb_u16 = binary.BigEndian.Uint16

func rb_u8(b []byte) uint8 {
	return b[0]
}

func rb_i64(b []byte) int64 {
	return int64(rb_u64(b))
}

func rb_i32(b []byte) int32 {
	return int32(rb_u32(b))
}

func rb_i16(b []byte) int16 {
	return int16(rb_u16(b))
}

func r_dstring(b []byte, fieldlen int) string {
	if fieldlen == 0 {
		return ""
	}
	return string(b[:b[fieldlen-1]])
}

func r_dcharacters(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	switch b[0] {
	case 8:
		s, _, err := transform.Bytes(charmap.Windows1252.NewDecoder(), b[1:])
		if err != nil {
			panic(err)
		}
		return string(s)
	case 16:
		s, _, err := transform.Bytes(unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder(), b[1:])
		if err != nil {
			panic(err)
		}
		return string(s)
	default:
		return ""
	}
}

func r_timestamp(b []byte) time.Time {
	var t time.Time
	t = t.AddDate(int(rl_u16(b[2:])), int(b[4]), int(b[5]))
	t.Add(time.Duration(b[6])*time.Hour +
		time.Duration(b[7])*time.Minute +
		time.Duration(b[8])*time.Second)
	return t
}
