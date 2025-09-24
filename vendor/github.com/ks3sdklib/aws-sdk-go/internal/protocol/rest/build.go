// Package rest provides RESTful serialisation of AWS requests and responses.
package rest

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/apierr"
	"io"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// RFC822 returns an RFC822 formatted timestamp for AWS protocols
const RFC822 = "Mon, 2 Jan 2006 15:04:05 GMT"

// Whether the byte value can be sent without escaping in AWS URLs
var noEscape [256]bool

func init() {
	for i := 0; i < len(noEscape); i++ {
		// AWS expects every character except these to be escaped
		noEscape[i] = (i >= 'A' && i <= 'Z') ||
			(i >= 'a' && i <= 'z') ||
			(i >= '0' && i <= '9') ||
			i == '-' ||
			i == '.' ||
			i == '_' ||
			i == '~'
	}
}

var isTokenTable = [256]bool{
	'!':  true,
	'#':  true,
	'$':  true,
	'%':  true,
	'&':  true,
	'\'': true,
	'*':  true,
	'+':  true,
	'-':  true,
	'.':  true,
	'0':  true,
	'1':  true,
	'2':  true,
	'3':  true,
	'4':  true,
	'5':  true,
	'6':  true,
	'7':  true,
	'8':  true,
	'9':  true,
	'A':  true,
	'B':  true,
	'C':  true,
	'D':  true,
	'E':  true,
	'F':  true,
	'G':  true,
	'H':  true,
	'I':  true,
	'J':  true,
	'K':  true,
	'L':  true,
	'M':  true,
	'N':  true,
	'O':  true,
	'P':  true,
	'Q':  true,
	'R':  true,
	'S':  true,
	'T':  true,
	'U':  true,
	'W':  true,
	'V':  true,
	'X':  true,
	'Y':  true,
	'Z':  true,
	'^':  true,
	'_':  true,
	'`':  true,
	'a':  true,
	'b':  true,
	'c':  true,
	'd':  true,
	'e':  true,
	'f':  true,
	'g':  true,
	'h':  true,
	'i':  true,
	'j':  true,
	'k':  true,
	'l':  true,
	'm':  true,
	'n':  true,
	'o':  true,
	'p':  true,
	'q':  true,
	'r':  true,
	's':  true,
	't':  true,
	'u':  true,
	'v':  true,
	'w':  true,
	'x':  true,
	'y':  true,
	'z':  true,
	'|':  true,
	'~':  true,
}

func ValidHeaderFieldName(v string) bool {
	if len(v) == 0 {
		return false
	}
	for i := 0; i < len(v); i++ {
		if !isTokenTable[v[i]] {
			return false
		}
	}
	return true
}

// Build builds the REST component of a service request.
func Build(r *aws.Request) {
	if r.ParamsFilled() {
		v := reflect.ValueOf(r.Params).Elem()
		buildLocationElements(r, v)
		buildBody(r, v)
	}
}

func buildLocationElements(r *aws.Request, v reflect.Value) {
	query := r.HTTPRequest.URL.Query()

	for i := 0; i < v.NumField(); i++ {
		m := v.Field(i)
		if n := v.Type().Field(i).Name; n[0:1] == strings.ToLower(n[0:1]) {
			continue
		}

		if m.IsValid() {
			field := v.Type().Field(i)
			name := field.Tag.Get("locationName")
			if name == "" {
				name = field.Name
			}
			if m.Kind() == reflect.Ptr {
				m = m.Elem()
			}
			if !m.IsValid() {
				continue
			}

			switch field.Tag.Get("location") {
			case "headers": // header maps
				buildHeaderMap(r, m, field.Tag.Get("locationName"))
			case "header":
				buildHeader(r, m, name)
			case "uri":
				buildURI(r, m, name)
			case "querystring":
				buildQueryString(r, m, name, query)
			case "querystrings":
				buildQueryStrings(r, m, name, query)
			}
		}
		if r.Error != nil {
			return
		}
	}

	buildExtendHeaders(r, v)
	buildExtendQueryParams(v, query)

	r.HTTPRequest.URL.RawQuery = query.Encode()
	updatePath(r.HTTPRequest.URL, r.Config)
}

func buildBody(r *aws.Request, v reflect.Value) {
	if field, ok := v.Type().FieldByName("SDKShapeTraits"); ok {
		if payloadName := field.Tag.Get("payload"); payloadName != "" {
			pfield, _ := v.Type().FieldByName(payloadName)
			if ptag := pfield.Tag.Get("type"); ptag != "" && ptag != "structure" {
				payload := reflect.Indirect(v.FieldByName(payloadName))
				if payload.IsValid() && payload.Interface() != nil {
					switch reader := payload.Interface().(type) {
					case io.ReadSeeker:
						r.SetReaderBody(reader)
					case []byte:
						r.SetBufferBody(reader)
					case string:
						r.SetStringBody(reader)
					default:
						r.Error = apierr.New("Marshal",
							"failed to encode REST request",
							fmt.Errorf("unknown payload type %s", payload.Type()))
					}
				}
			}
		}
	}
}

func buildHeader(r *aws.Request, v reflect.Value, name string) {
	str, err := convertType(v)
	if err != nil {
		r.Error = apierr.New("Marshal", "failed to encode REST request", err)
	} else if str != nil {
		r.HTTPRequest.Header.Add(name, *str)
	}
}

func buildHeaderMap(r *aws.Request, v reflect.Value, prefix string) {
	for _, key := range v.MapKeys() {
		str, err := convertType(v.MapIndex(key))
		if err != nil {
			r.Error = apierr.New("Marshal", "failed to encode REST request", err)
		} else if str != nil {
			if strings.HasPrefix(strings.ToLower(key.String()), strings.ToLower(prefix)) {
				r.HTTPRequest.Header.Add(key.String(), *str)
			} else {
				r.HTTPRequest.Header.Add(prefix+key.String(), *str)
			}
		}
	}
}

func buildExtendHeaders(r *aws.Request, v reflect.Value) {
	extendHeaders := v.FieldByName("ExtendHeaders")
	if extendHeaders.IsValid() {
		iter := extendHeaders.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			value := iter.Value()
			if ValidHeaderFieldName(key) {
				if !value.IsNil() {
					r.HTTPRequest.Header.Set(key, value.Elem().String())
				} else {
					r.HTTPRequest.Header.Set(key, "")
				}
			} else {
				r.Error = apierr.New("Marshal", fmt.Sprintf("invalid extend header field name \"%s\"", key), nil)
				return
			}
		}
	}
}

func buildExtendQueryParams(v reflect.Value, query url.Values) {
	extendQueryParams := v.FieldByName("ExtendQueryParams")
	if extendQueryParams.IsValid() {
		iter := extendQueryParams.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			if key == "" {
				continue
			}
			value := iter.Value()
			if !value.IsNil() {
				query.Set(key, value.Elem().String())
			} else {
				query.Set(key, "")
			}
		}
	}
}

func buildURI(r *aws.Request, v reflect.Value, name string) {
	value, err := convertType(v)
	if err != nil {
		r.Error = apierr.New("Marshal", "failed to encode REST request", err)
	} else if value != nil {
		uri := r.HTTPRequest.URL.Path
		uri = strings.Replace(uri, "{"+name+"}", EscapePath(*value, true), -1)
		uri = strings.Replace(uri, "{"+name+"+}", EscapePath(*value, false), -1)
		r.HTTPRequest.URL.Path = uri
	}
}

func buildQueryString(r *aws.Request, v reflect.Value, name string, query url.Values) {
	str, err := convertType(v)
	if err != nil {
		r.Error = apierr.New("Marshal", "failed to encode REST request", err)
	} else if str != nil {
		query.Set(name, *str)
	} else if str == nil {
		query.Set(name, "")
	}
}

func buildQueryStrings(r *aws.Request, v reflect.Value, name string, query url.Values) {
	if v.Kind() == reflect.Slice {
		var valSlice []string
		for i := 0; i < v.Len(); i++ {
			str, err := convertType(v.Index(i))
			if err != nil {
				r.Error = apierr.New("Marshal", "failed to encode REST request", err)
				return
			}
			valSlice = append(valSlice, *str)
		}
		if len(valSlice) > 0 {
			query.Set(name, strings.Join(valSlice, ","))
		}
	}
}

func updatePath(url *url.URL, cfg *aws.Config) {
	urlPath := url.Path
	scheme, query := url.Scheme, url.RawQuery

	// path.Clean will remove duplicate leading /
	// this will make deleting / started key impossible
	// so escape it here first
	urlPath = strings.Replace(urlPath, "//", "/%2F", -1)

	// 新增参数控制path clean，默认值为true
	if !cfg.DisableRestProtocolURICleaning {
		urlPath = cleanPath(urlPath)
	}

	// get formatted URL minus scheme, so we can build this into Opaque
	url.Scheme, url.Path, url.RawQuery = "", "", ""
	s := url.String()
	url.Scheme = scheme
	url.RawQuery = query

	// build opaque URI
	url.Opaque = s + urlPath
}

func cleanPath(urlPath string) string {
	// path.Clean会去掉最后的斜杠，导致无法创建目录。所以添加以下逻辑
	add := false
	if urlPath[len(urlPath)-1] == '/' && len(urlPath) > 1 {
		add = true
	}

	// clean up path
	urlPath = path.Clean(urlPath)
	if add {
		urlPath += "/"
	}

	return urlPath
}

// EscapePath escapes part of a URL path in Amazon style
//
// path The path segment to escape
// encodeSep If true, '/' will be encoded, otherwise they will not
func EscapePath(path string, encodeSep bool) string {
	var buf bytes.Buffer
	for i := 0; i < len(path); i++ {
		c := path[i]
		if noEscape[c] || (c == '/' && !encodeSep) {
			buf.WriteByte(c)
		} else {
			fmt.Fprintf(&buf, "%%%02X", c)
		}
	}
	return buf.String()
}

func convertType(v reflect.Value) (*string, error) {
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return nil, nil
	}

	var str string
	switch value := v.Interface().(type) {
	case string:
		str = value
	case []byte:
		str = base64.StdEncoding.EncodeToString(value)
	case bool:
		str = strconv.FormatBool(value)
	case int64:
		str = strconv.FormatInt(value, 10)
	case float64:
		str = strconv.FormatFloat(value, 'f', -1, 64)
	case time.Time:
		str = value.UTC().Format(RFC822)
	default:
		err := fmt.Errorf("unsupported value for param %v (%s)", v.Interface(), v.Type())
		return nil, err
	}
	return &str, nil
}
