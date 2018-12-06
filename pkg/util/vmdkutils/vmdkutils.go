package vmdkutils

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/utils"
)

type SVMDKInfo struct {
	ExtentFile       string
	Heads            int64
	Sectors          int64
	Cylinders        int64
	CID              string
	LongCID          string
	UUID             string
	AdapterType      string
	VirtualHWVersion string
}

const (
	extentPatternString = `^RW \d+ VMFS\w* \"(?P<fn>[^"]+)`
)

var (
	extentPatternRegexp = regexp.MustCompile(extentPatternString)
)

func Parse(content string) (*SVMDKInfo, error) {
	return ParseStream(strings.NewReader(content))
}

func ParseStream(stream io.Reader) (*SVMDKInfo, error) {
	info := SVMDKInfo{}
	scanner := bufio.NewScanner(stream)
	findExtent := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := extentPatternRegexp.FindStringSubmatch(line)
		if len(matches) > 0 {
			// log.Debugf("%#v", matches)
			info.ExtentFile = matches[1]
			findExtent = true
		} else {
			equalPos := strings.IndexByte(line, '=')
			if equalPos > 0 {
				key := strings.TrimSpace(line[:equalPos])
				value := utils.Unquote(strings.TrimSpace(line[equalPos+1:]))
				switch key {
				case "CID":
					info.CID = value
				case "ddb.uuid":
					info.UUID = value
				case "ddb.geometry.cylinders":
					info.Cylinders, _ = strconv.ParseInt(value, 10, 64)
				case "ddb.geometry.heads":
					info.Heads, _ = strconv.ParseInt(value, 10, 64)
				case "ddb.geometry.sectors":
					info.Sectors, _ = strconv.ParseInt(value, 10, 64)
				case "ddb.longContentID":
					info.LongCID = value
				case "ddb.adapterType":
					info.AdapterType = value
				case "ddb.virtualHWVersion":
					info.VirtualHWVersion = value
				}
			}
		}
	}
	if !findExtent {
		return nil, fmt.Errorf("not a vmdk file")
	}
	return &info, nil
}
