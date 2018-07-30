package fileutils

import (
	"fmt"
	"strconv"

	"github.com/yunionio/pkg/util/regutils"
)

func parseSizeStr(sizeStr string, defaultUnit byte, base int) (int, error) {
	var numPart int
	var unit byte
	if regutils.MatchInteger(sizeStr) {
		numPart, _ = strconv.Atoi(sizeStr)
		unit = defaultUnit
	} else if regutils.MatchSize(sizeStr) {
		numPart, _ = strconv.Atoi(sizeStr[:len(sizeStr)-1])
		unit = sizeStr[len(sizeStr)-1]
	} else {
		return 0, fmt.Errorf("Invalid size string %s", sizeStr)
	}
	switch unit {
	case 'g', 'G':
		return numPart * base * base * base, nil
	case 'm', 'M':
		return numPart * base * base, nil
	case 'k', 'K':
		return numPart * base, nil
	case 't', 'T':
		return numPart * base * base * base * base, nil
	case 'p', 'P':
		return numPart * base * base * base * base * base, nil
	default:
		return numPart, nil
	}
}

func GetSizeGb(sizeStr string, defaultSize byte, base int) (int, error) {
	size, err := parseSizeStr(sizeStr, defaultSize, base)
	return size / base / base / base, err
}

func GetSizeMb(sizeStr string, defaultSize byte, base int) (int, error) {
	size, err := parseSizeStr(sizeStr, defaultSize, base)
	return size / base / base, err
}

func GetSizeKb(sizeStr string, defaultSize byte, base int) (int, error) {
	size, err := parseSizeStr(sizeStr, defaultSize, base)
	return size / base, err
}
