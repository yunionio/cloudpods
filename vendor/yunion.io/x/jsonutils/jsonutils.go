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

package jsonutils

import (
	"bytes"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/sortedmap"
)

// swagger:type object
type JSONObject interface {
	gotypes.ISerializable

	parse(s *sJsonParseSession, str []byte, offset int) (int, error)
	writeSource

	// String() string
	PrettyString() string
	prettyString(level int) string
	YAMLString() string
	QueryString() string
	_queryString(key string) string
	Contains(keys ...string) bool
	ContainsIgnoreCases(keys ...string) bool
	Get(keys ...string) (JSONObject, error)
	GetIgnoreCases(keys ...string) (JSONObject, error)
	GetAt(i int, keys ...string) (JSONObject, error)
	Int(keys ...string) (int64, error)
	Float(keys ...string) (float64, error)
	Bool(keys ...string) (bool, error)
	GetMap(keys ...string) (map[string]JSONObject, error)
	GetArray(keys ...string) ([]JSONObject, error)
	GetTime(keys ...string) (time.Time, error)
	GetString(keys ...string) (string, error)
	Unmarshal(obj interface{}, keys ...string) error
	Equals(obj JSONObject) bool
	unmarshalValue(s *sJsonUnmarshalSession, val reflect.Value) error
	// IsZero() bool
	Interface() interface{}
	isCompond() bool
}

type JSONValue struct {
}

var (
	JSONNull  = &JSONValue{}
	JSONTrue  = &JSONBool{data: true}
	JSONFalse = &JSONBool{data: false}
)

// swagger:type object
type JSONDict struct {
	JSONValue
	data sortedmap.SSortedMap

	nodeId int
}

type JSONArray struct {
	JSONValue
	data []JSONObject
}

type JSONString struct {
	JSONValue
	data string
}

type JSONInt struct {
	JSONValue
	data int64
}

type JSONFloat struct {
	JSONValue
	data float64
	bit  int
}

type JSONBool struct {
	JSONValue
	data bool
}

func skipEmpty(str []byte, offset int) int {
	i := offset
	for i < len(str) {
		switch str[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

// isFiniteFloat reports whether the value has a json representation
func isFiniteFloat(val float64) bool {
	return !math.IsNaN(val) && !math.IsInf(val, 0)
}

func hexchar2num(v byte) (byte, error) {
	switch {
	case v >= '0' && v <= '9':
		return v - '0', nil
	case v >= 'a' && v <= 'f':
		return v - 'a' + 10, nil
	case v >= 'A' && v <= 'F':
		return v - 'A' + 10, nil
	default:
		return 0, ErrInvalidChar // fmt.Errorf("Illegal char %c", v)
	}
}

func hexstr2byte(str []byte) (byte, error) {
	if len(str) < 2 {
		return 0, ErrInvalidHex // fmt.Errorf("Input must be 2 hex chars")
	}
	v1, e := hexchar2num(str[0])
	if e != nil {
		return 0, e
	}
	v2, e := hexchar2num(str[1])
	if e != nil {
		return 0, e
	}
	return v1*16 + v2, nil
}

func hexstr2rune(str []byte) (rune, error) {
	if len(str) < 4 {
		return 0, ErrInvalidRune // fmt.Errorf("Input must be 4 hex chars")
	}
	v1, e := hexstr2byte(str[0:2])
	if e != nil {
		return 0, e
	}
	v2, e := hexstr2byte(str[2:4])
	if e != nil {
		return 0, e
	}
	return rune(v1)*256 + rune(v2), nil
}

func parseQuoteString(str []byte, offset int, quotec byte) (string, int, error) {
	var (
		buffer    []byte
		runebytes = make([]byte, 4)
		runen     int
		i         = offset
	)
ret:
	for i < len(str) {
		switch str[i] {
		case '\\':
			if i+1 < len(str) {
				i++
				switch str[i] {
				case 'u':
					i++
					if i+4 >= len(str) {
						return "", i, NewJSONError(str, i, "Incomplete unicode")
					}
					r, e := hexstr2rune(str[i : i+4])
					if e != nil {
						return "", i, NewJSONError(str, i, e.Error())
					}
					i += 4
					if utf16.IsSurrogate(r) {
						// a character outside the BMP is written as a
						// surrogate pair, e.g. 😀
						if i+6 <= len(str) && str[i] == '\\' && str[i+1] == 'u' {
							r2, e2 := hexstr2rune(str[i+2 : i+6])
							if e2 == nil {
								if combined := utf16.DecodeRune(r, r2); combined != utf8.RuneError {
									r = combined
									i += 6
								}
							}
						}
					}
					runen = utf8.EncodeRune(runebytes, r)
					buffer = append(buffer, runebytes[0:runen]...)
				case 'x':
					i++
					if i+2 >= len(str) {
						return "", i, NewJSONError(str, i, "Incomplete hex")
					}
					b, e := hexstr2byte(str[i : i+2])
					if e != nil {
						return "", i, NewJSONError(str, i, e.Error())
					}
					buffer = append(buffer, b)
					i += 2
				case 'n':
					buffer = append(buffer, '\n')
					i++
				case 'r':
					buffer = append(buffer, '\r')
					i++
				case 't':
					buffer = append(buffer, '\t')
					i++
				case 'b':
					buffer = append(buffer, '\b')
					i++
				case 'f':
					buffer = append(buffer, '\f')
					i++
				case '\\':
					buffer = append(buffer, '\\')
					i++
				default:
					buffer = append(buffer, str[i])
					i++
				}
			} else {
				return "", i, NewJSONError(str, i, "Incomplete escape")
			}
		case quotec:
			i++
			break ret
		default:
			buffer = append(buffer, str[i])
			i++
		}
	}
	return string(buffer), i, nil
}

func parseString(str []byte, offset int) (string, bool, int, error) {
	var (
		i = offset
	)
	if c := str[i]; c == '"' || c == '\'' {
		r, newOfs, err := parseQuoteString(str, i+1, c)
		return r, true, newOfs, err
	}
ret2:
	for i < len(str) {
		switch str[i] {
		case ' ', ':', ',', '\t', '\r', '\n', '}', ']':
			break ret2
		default:
			i++
		}
	}
	return string(str[offset:i]), false, i, nil
}

// isNodeReference reports whether the token has the form of a node
// reference, that is a bare <N> with an integer N
func isNodeReference(val string) bool {
	if len(val) < 3 || val[0] != '<' || val[len(val)-1] != '>' {
		return false
	}
	_, err := strconv.ParseInt(val[1:len(val)-1], 10, 64)
	return err == nil
}

func (s *sJsonParseSession) parseJSONValue(str []byte, offset int) (JSONObject, int, error) {
	val, quote, i, e := parseString(str, offset)
	if e != nil {
		return nil, i, errors.Wrap(e, "parseString")
	} else if quote {
		return &JSONString{data: val}, i, nil
	} else if s.allowNodeReference && len(val) > 1 && val[0] == '<' && val[len(val)-1] == '>' {
		// Pointer <nnnn>
		val = val[1 : len(val)-1]
		ival, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, i, errors.Wrapf(errors.ErrInvalidStatus, "invalid node id %s", val)
		}
		nodeId := int(ival)
		ptr := &sJSONPointer{
			nodeId: nodeId,
		}
		s.saveReferer(nodeId, ptr)
		return ptr, i, nil
	} else if !s.allowNodeReference && isNodeReference(val) {
		// a node reference can not be left unresolved and kept as a plain
		// value: a caller could not tell it apart from a real string
		return nil, i, errors.Wrap(ErrNodeReferenceDisabled, val)
	} else {
		lval := strings.ToLower(val)
		if len(lval) == 0 || lval == "null" || lval == "none" {
			return JSONNull, i, nil
		}
		if lval == "true" || lval == "yes" {
			return JSONTrue, i, nil
		}
		if lval == "false" || lval == "no" {
			return JSONFalse, i, nil
		}
		ival, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			return &JSONInt{data: ival}, i, nil
		}
		fval, err := strconv.ParseFloat(val, 64)
		if err == nil && isFiniteFloat(fval) {
			return &JSONFloat{data: fval}, i, nil
		}
		// nan and +-inf have no json representation, keep them as strings
		return &JSONString{data: val}, i, nil
	}
}

// https://www.ietf.org/rfc/rfc4627.txt
//
//	string = quotation-mark *char quotation-mark
//
//	char = unescaped /
//	       escape (
//	           %x22 /          ; "    quotation mark  U+0022
//	           %x5C /          ; \    reverse solidus U+005C
//	           %x2F /          ; /    solidus         U+002F
//	           %x62 /          ; b    backspace       U+0008
//	           %x66 /          ; f    form feed       U+000C
//	           %x6E /          ; n    line feed       U+000A
//	           %x72 /          ; r    carriage return U+000D
//	           %x74 /          ; t    tab             U+0009
//	           %x75 4HEXDIG )  ; uXXXX                U+XXXX
//
//	escape = %x5C              ; \
//
//	quotation-mark = %x22      ; "
//
//	unescaped = %x20-21 / %x23-5B / %x5D-10FFFF
func escapeJsonChar(sb *strings.Builder, ch byte) {
	switch ch {
	case '"':
		sb.Write([]byte{'\\', '"'})
	case '\\':
		sb.Write([]byte{'\\', '\\'})
	case '\b':
		sb.Write([]byte{'\\', 'b'})
	case '\f':
		sb.Write([]byte{'\\', 'f'})
	case '\n':
		sb.Write([]byte{'\\', 'n'})
	case '\r':
		sb.Write([]byte{'\\', 'r'})
	case '\t':
		sb.Write([]byte{'\\', 't'})
	default:
		// RFC 8259: U+0000–U+001F must be escaped as \uXXXX.
		if ch < 0x20 {
			const hexdigits = "0123456789abcdef"
			sb.Write([]byte{'\\', 'u', '0', '0', hexdigits[ch>>4], hexdigits[ch&0xf]})
			return
		}
		sb.WriteByte(ch)
		/*if ((ch >= 0x20 && ch <= 0x21) || (ch >= 0x23 || ch <= 0x5B) || (ch >= 0x5D && ch <= 0x10FFFF)) && ch != 0x81 && ch != 0x8d && ch != 0x8f && ch != 0x90 && ch != 0x9d {
			sb.WriteRune(ch)
		} else if ch <= 0xff {
			sb.Write([]byte{'\\', 'x'})
			sb.WriteString(fmt.Sprintf("%02x", ch))
		} else if ch <= 0xffff {
			sb.Write([]byte{'\\', 'u'})
			sb.WriteString(fmt.Sprintf("%04x", ch))
		} else {
			sb.Write([]byte{'\\', 'u'})
			sb.WriteString(fmt.Sprintf("%04x", ch>>16))
			sb.Write([]byte{'\\', 'u'})
			sb.WriteString(fmt.Sprintf("%04x", (ch & 0xffff)))
		}*/
	}
}

// escapeJsonByte writes a byte that is not part of a valid utf-8 sequence
// as a \xXX escape, which parseQuoteString reads back unchanged
func escapeJsonByte(sb *strings.Builder, ch byte) {
	const hexdigits = "0123456789abcdef"
	sb.Write([]byte{'\\', 'x', hexdigits[ch>>4], hexdigits[ch&0xf]})
}

func quoteString(str string) string {
	sb := &strings.Builder{}
	sb.Grow(len(str) + 2)
	sb.WriteByte('"')
	for i := 0; i < len(str); {
		ch := str[i]
		if ch < utf8.RuneSelf {
			escapeJsonChar(sb, ch)
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(str[i:])
		if r == utf8.RuneError && size <= 1 {
			// keep a non utf-8 byte reversible instead of writing it out
			escapeJsonByte(sb, ch)
			i++
			continue
		}
		sb.WriteString(str[i : i+size])
		i += size
	}
	sb.WriteByte('"')
	return sb.String()
}

func jsonPrettyString(o JSONObject, level int) string {
	var buffer bytes.Buffer
	for i := 0; i < level; i++ {
		buffer.WriteString("  ")
	}
	buffer.WriteString(o.String())
	return buffer.String()
}

func (this *JSONString) PrettyString() string {
	return this.String()
}

func (this *JSONString) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONValue) parse(s *sJsonParseSession, str []byte, offset int) (int, error) {
	return 0, nil
}

func (this *JSONValue) PrettyString() string {
	return this.String()
}

func (this *JSONValue) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONInt) PrettyString() string {
	return this.String()
}

func (this *JSONInt) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONFloat) PrettyString() string {
	return this.String()
}

func (this *JSONFloat) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONBool) PrettyString() string {
	return this.String()
}

func (this *JSONBool) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (s *sJsonParseSession) parseDict(str []byte, offset int) (sortedmap.SSortedMap, int, int, error) {
	var nodeId int
	smap := sortedmap.NewSortedMap()
	if str[offset] != '{' {
		return smap, offset, nodeId, NewJSONError(str, offset, "{ not found")
	}
	var i = offset + 1
	var e error = nil
	var key string
	var stop = false
	// collect the keys first so that the sorted map can be built in key
	// order: adding to a sorted map out of order shifts the whole tail
	// on every insert
	values := make(map[string]JSONObject)
	keys := make([]string, 0)
	for !stop && i < len(str) {
		i = skipEmpty(str, i)
		if i >= len(str) {
			return smap, i, nodeId, NewJSONError(str, i, "Truncated")
		}
		if str[i] == '}' {
			stop = true
			i++
			continue
		}
		key, _, i, e = parseString(str, i)
		if e != nil {
			return smap, i, nodeId, errors.Wrap(e, "parseString")
		}
		if i >= len(str) {
			return smap, i, nodeId, NewJSONError(str, i, "Truncated")
		}
		i = skipEmpty(str, i)
		if i >= len(str) {
			return smap, i, nodeId, NewJSONError(str, i, "Truncated")
		}
		if str[i] != ':' {
			return smap, i, nodeId, NewJSONError(str, i, ": not found")
		}
		i++
		i = skipEmpty(str, i)
		if i >= len(str) {
			return smap, i, nodeId, NewJSONError(str, i, "Truncated")
		}
		var val JSONObject = nil
		switch str[i] {
		case '[':
			val = &JSONArray{}
			i, e = val.parse(s, str, i)
		case '{':
			val = &JSONDict{}
			i, e = val.parse(s, str, i)
		default:
			val, i, e = s.parseJSONValue(str, i)
		}
		if e != nil {
			return smap, i, nodeId, errors.Wrap(e, "parse misc")
		}
		if s.allowNodeReference && key == jsonPointerKey {
			// node id
			jval, ok := val.(*JSONInt)
			if !ok {
				return smap, i, nodeId, errors.Wrap(ErrInvalidNodeId, jsonPointerKey)
			}
			nodeId = int(jval.data)
		} else {
			if _, ok := values[key]; !ok {
				keys = append(keys, key)
			}
			values[key] = val
		}
		i = skipEmpty(str, i)
		if i >= len(str) {
			return smap, i, nodeId, NewJSONError(str, i, "Truncated")
		}
		switch str[i] {
		case ',':
			i++
		case '}':
			i++
			stop = true
		default:
			return smap, i, nodeId, NewJSONError(str, i, "Unexpected char")
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		smap = sortedmap.Add(smap, key, values[key])
	}
	return smap, i, nodeId, nil
}

func (s *sJsonParseSession) parseArray(str []byte, offset int) ([]JSONObject, int, error) {
	if str[offset] != '[' {
		return nil, offset, NewJSONError(str, offset, "[ not found")
	}
	var (
		list []JSONObject
		i    = offset + 1
		val  JSONObject
		e    error
		stop bool
	)
	for !stop && i < len(str) {
		i = skipEmpty(str, i)
		if i >= len(str) {
			return list, i, NewJSONError(str, i, "Truncated")
		}
		switch str[i] {
		case ']':
			i++
			stop = true
			continue
		case '[':
			val = &JSONArray{}
			i, e = val.parse(s, str, i)
		case '{':
			val = &JSONDict{}
			i, e = val.parse(s, str, i)
		default:
			val, i, e = s.parseJSONValue(str, i)
		}
		if e != nil {
			return list, i, errors.Wrap(e, "parse misc")
		}
		if i >= len(str) {
			return list, i, NewJSONError(str, i, "Truncated")
		}
		list = append(list, val)
		i = skipEmpty(str, i)
		if i >= len(str) {
			return list, i, NewJSONError(str, i, "Truncated")
		}
		switch str[i] {
		case ',':
			i++
		case ']':
			i++
			stop = true
		default:
			return list, i, NewJSONError(str, i, "Unexpected char")
		}
	}
	return list, i, nil
}

func (this *JSONDict) parse(s *sJsonParseSession, str []byte, offset int) (int, error) {
	e := s.enter()
	if e != nil {
		return offset, errors.Wrap(e, "enter")
	}
	defer s.leave()
	smap, i, nodeId, e := s.parseDict(str, offset)
	if e == nil {
		this.nodeId = nodeId
		this.data = smap
		if this.nodeId > 0 {
			e = s.saveNode(nodeId, this)
			if e != nil {
				return i, errors.Wrap(e, "saveNode")
			}
		}
		return i, nil
	}
	return i, errors.Wrap(e, "parseDict")
}

func (this *JSONDict) SortedKeys() []string {
	return this.data.Keys()
}

func (this *JSONDict) PrettyString() string {
	return this.prettyString(0)
}

func (this *JSONDict) prettyString(level int) string {
	var buffer bytes.Buffer
	var linebuf bytes.Buffer
	for i := 0; i < level; i++ {
		linebuf.WriteString("  ")
	}
	var tab = linebuf.String()
	buffer.WriteString(tab)
	buffer.WriteByte('{')
	var idx = 0
	for iter := sortedmap.NewIterator(this.data); iter.HasMore(); iter.Next() {
		k, vInf := iter.Get()
		v := vInf.(JSONObject)
		if idx > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteByte('\n')
		buffer.WriteString(tab)
		buffer.WriteString("  ")
		buffer.WriteString(quoteString(k))
		buffer.WriteByte(':')
		if gotypes.IsNil(v) {
			buffer.WriteByte(' ')
			buffer.WriteString("null")
		} else {
			_, okdict := v.(*JSONDict)
			_, okarray := v.(*JSONArray)
			if okdict || okarray {
				buffer.WriteByte('\n')
				buffer.WriteString(v.prettyString(level + 2))
			} else {
				buffer.WriteByte(' ')
				buffer.WriteString(v.String())
			}
		}
		idx++
	}
	if len(this.data) > 0 {
		buffer.WriteByte('\n')
		buffer.WriteString(tab)
	}
	buffer.WriteByte('}')
	return buffer.String()
}

func (this *JSONArray) parse(s *sJsonParseSession, str []byte, offset int) (int, error) {
	e := s.enter()
	if e != nil {
		return offset, errors.Wrap(e, "enter")
	}
	defer s.leave()
	val, i, e := s.parseArray(str, offset)
	if e == nil {
		this.data = val
	}
	return i, errors.Wrap(e, "parseArray")
}

func (this *JSONArray) PrettyString() string {
	return this.prettyString(0)
}

func (this *JSONArray) prettyString(level int) string {
	var buffer bytes.Buffer
	var linebuf bytes.Buffer
	for i := 0; i < level; i++ {
		linebuf.WriteString("  ")
	}
	var tab = linebuf.String()
	buffer.WriteString(tab)
	buffer.WriteByte('[')
	for idx, v := range this.data {
		if idx > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteByte('\n')
		if gotypes.IsNil(v) {
			buffer.WriteString(tab)
			buffer.WriteString("  null")
		} else {
			buffer.WriteString(v.prettyString(level + 1))
		}
	}
	if len(this.data) > 0 {
		buffer.WriteByte('\n')
		buffer.WriteString(tab)
	}
	buffer.WriteByte(']')
	return buffer.String()
}

func ParseString(str string) (JSONObject, error) {
	return Parse([]byte(str))
}

func Parse(str []byte) (JSONObject, error) {
	json, offset, err := ParseStream(str, 0)
	if err != nil {
		return nil, err
	}
	if i := skipEmpty(str, offset); i < len(str) {
		return nil, NewJSONError(str, i, "Unexpected content after the value")
	}
	return json, nil
}

// ParseStream parses one value starting at offset, it returns the value and
// the offset just after it, so that a stream of concatenated values can be
// walked.  Node references are not resolved, see ParseTrusted.
func ParseStream(str []byte, offset int) (JSONObject, int, error) {
	return parseStream(str, offset, false)
}

// ParseTrusted parses a document from a trusted source, resolving the node
// references that Marshal writes for a cyclic object: the ___jnid_ key inside
// an object and a bare <N> value referring to it.
//
// A document from an untrusted source must be parsed with Parse instead.  A
// resolved reference makes two fields of the target struct point at the same
// object, which is what the round trip of a cyclic object needs, but it also
// lets a forged document do the same.
//
// Note that Marshal needs this syntax to terminate on a cyclic object, so it
// keeps writing it either way.
func ParseTrusted(str []byte) (JSONObject, error) {
	json, _, err := parseStream(str, 0, true)
	return json, err
}

// ParseTrustedString is ParseTrusted for a string
func ParseTrustedString(str string) (JSONObject, error) {
	return ParseTrusted([]byte(str))
}

func parseStream(str []byte, offset int, allowNodeReference bool) (JSONObject, int, error) {
	s := newJsonParseSession(allowNodeReference)
	i := offset
	i = skipEmpty(str, i)
	var val JSONObject = nil
	var e error = nil
	if i < len(str) {
		switch str[i] {
		case '{':
			val = &JSONDict{}
			i, e = val.parse(s, str, i)
		case '[':
			val = &JSONArray{}
			i, e = val.parse(s, str, i)
		default:
			val, i, e = s.parseJSONValue(str, i)
			// return nil, NewJSONError(str, i, "Invalid JSON string")
		}
		if e != nil {
			return nil, i, errors.Wrap(e, "parse misc")
		} else {
			return val, i, nil
		}
	} else {
		return nil, i, NewJSONError(str, i, "Empty string")
	}
}

func ParseJsonStreams(stream []byte) ([]JSONObject, error) {
	ret := make([]JSONObject, 0)
	errs := make([]error, 0)
	offset := 0
	for offset < len(stream) {
		for offset < len(stream) && stream[offset] != '[' && stream[offset] != '{' {
			offset++
		}
		if offset >= len(stream) {
			break
		}
		json, noffset, err := ParseStream(stream, offset)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "jsonutils.ParseStream fail at %d", offset))
			offset++
		} else {
			ret = append(ret, json)
			offset = noffset
		}
	}
	if len(errs) > 0 && len(ret) == 0 {
		return nil, errors.NewAggregate(errs)
	}
	if len(errs) > 0 {
		log.Warningf("jsonutils.ParseJsonStreams: %d errors, %s", len(errs), errors.NewAggregate(errs))
	}
	return ret, nil
}
