package awsutil

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"github.com/ks3sdklib/aws-sdk-go/internal/protocol/xml/xmlutil"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

// GoFmt returns the Go formated string of the input.
//
// Panics if the format fails.
func GoFmt(buf string) string {
	formatted, err := format.Source([]byte(buf))
	if err != nil {
		panic(fmt.Errorf("%s\nOriginal code:\n%s", err.Error(), buf))
	}
	return string(formatted)
}

var reTrim = regexp.MustCompile(`\s{2,}`)

// Trim removes all leading and trailing white space.
//
// All consecutive spaces will be reduced to a single space.
func Trim(s string) string {
	return strings.TrimSpace(reTrim.ReplaceAllString(s, " "))
}

// Capitalize capitalizes the first character of the string.
func Capitalize(s string) string {
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

// SortXML sorts the reader's XML elements
func SortXML(r io.Reader) string {
	var buf bytes.Buffer
	d := xml.NewDecoder(r)
	root, _ := xmlutil.XMLToStruct(d, nil)
	e := xml.NewEncoder(&buf)
	xmlutil.StructToXML(e, root, true)
	return buf.String()
}

// PrettyPrint generates a human readable representation of the value v.
// All values of v are recursively found and pretty printed also.
func PrettyPrint(v interface{}) string {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Struct:
		str := fullName(value.Type()) + "{\n"
		for i := 0; i < value.NumField(); i++ {
			l := string(value.Type().Field(i).Name[0])
			if strings.ToUpper(l) == l {
				str += value.Type().Field(i).Name + ": "
				str += PrettyPrint(value.Field(i).Interface())
				str += ",\n"
			}
		}
		str += "}"
		return str
	case reflect.Map:
		str := "map[" + fullName(value.Type().Key()) + "]" + fullName(value.Type().Elem()) + "{\n"
		for _, k := range value.MapKeys() {
			str += "\"" + k.String() + "\": "
			str += PrettyPrint(value.MapIndex(k).Interface())
			str += ",\n"
		}
		str += "}"
		return str
	case reflect.Ptr:
		if e := value.Elem(); e.IsValid() {
			return "&" + PrettyPrint(e.Interface())
		}
		return "nil"
	case reflect.Slice:
		str := "[]" + fullName(value.Type().Elem()) + "{\n"
		for i := 0; i < value.Len(); i++ {
			str += PrettyPrint(value.Index(i).Interface())
			str += ",\n"
		}
		str += "}"
		return str
	default:
		return fmt.Sprintf("%#v", v)
	}
}

func pkgName(t reflect.Type) string {
	pkg := t.PkgPath()
	c := strings.Split(pkg, "/")
	return c[len(c)-1]
}

func fullName(t reflect.Type) string {
	if pkg := pkgName(t); pkg != "" {
		return pkg + "." + t.Name()
	}
	return t.Name()
}

//获取指定目录及所有子目录下的所有文件，可以匹配后缀过滤。
func WalkDir(dirPth, suffix string) (files []string, err error) {
	files = make([]string, 0, 30)
	suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写

	err = filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error { //遍历目录
		//if err != nil { //忽略错误
		// return err
		//}

		if fi.IsDir() { // 忽略目录
			return nil
		}

		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, filename)
		}

		return nil
	})
	return files, err
}
func IsHidden(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.Name()[0] == '.'
}
func ComputeMD5Hash(input []byte) []byte {
	h := md5.New()
	h.Write(input)
	return h.Sum(nil)
}

func EncodeAsString(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}
