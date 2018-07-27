package uuid

import (
	"encoding/hex"

	"github.com/golang-plus/errors"
)

// Parse parses the UUID string.
func Parse(str string) (UUID, error) {
	length := len(str)
	buffer := make([]byte, 16)
	indexes := []int{}
	switch length {
	case 36:
		if str[8] != '-' || str[13] != '-' || str[18] != '-' || str[23] != '-' {
			return Nil, errors.Newf("format of UUID string %q is invalid, it should be xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (8-4-4-4-12)", str)
		}
		indexes = []int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}
	case 32:
		indexes = []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30}
	default:
		return Nil, errors.Newf("length of UUID string %q is invalid, it should be 36 (standard) or 32 (without dash)", str)
	}

	var err error
	for i, v := range indexes {
		if c, e := hex.DecodeString(str[v : v+2]); e == nil {
			buffer[i] = c[0]
		} else {
			err = e
			break
		}
	}

	if err != nil {
		return Nil, errors.Wrapf(err, "UUID string %q is invalid", str)
	}

	uuid := UUID{}
	copy(uuid[:], buffer)

	if !uuid.Equal(Nil) {
		if uuid.Layout() == LayoutInvalid {
			return Nil, errors.Newf("layout of UUID %q is invalid", str)
		}

		if uuid.Version() == VersionUnknown {
			return Nil, errors.Newf("version of UUID %q is unknown", str)
		}
	}

	return uuid, nil
}

// IsValid reports whether the passed string is a valid uuid string.
func IsValid(uuid string) bool {
	_, err := Parse(uuid)
	return err == nil
}
