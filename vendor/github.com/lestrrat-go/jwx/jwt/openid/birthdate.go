package openid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

// https://openid.net/specs/openid-connect-core-1_0.html
//
// End-User's birthday, represented as an ISO 8601:2004 [ISO8601‑2004] YYYY-MM-DD format.
// The year MAY be 0000, indicating that it is omitted. To represent only the year, YYYY
// format is allowed. Note that depending on the underlying platform's date related function,
// providing just year can result in varying month and day, so the implementers need to
// take this factor into account to correctly process the dates.

type BirthdateClaim struct {
	year  *int
	month *int
	day   *int
}

func (b BirthdateClaim) Year() int {
	if b.year == nil {
		return 0
	}
	return *(b.year)
}

func (b BirthdateClaim) Month() int {
	if b.month == nil {
		return 0
	}
	return *(b.month)
}

func (b BirthdateClaim) Day() int {
	if b.day == nil {
		return 0
	}
	return *(b.day)
}

func (b *BirthdateClaim) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errors.Wrap(err, `failed to unmarshal JSON string for birthdate claim`)
	}

	if err := b.Accept(s); err != nil {
		return errors.Wrap(err, `failed to accept JSON value for birthdate claim`)
	}
	return nil
}

var birthdateRx = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)

// Accepts a value read from JSON, and converts it to a BirthdateClaim.
// This method DOES NOT verify the correctness of a date.
// Consumers should check for validity of dates such as Apr 31 et al
func (b *BirthdateClaim) Accept(v interface{}) error {
	switch v := v.(type) {
	case string:
		// yeah, yeah, regexp is slow. PR's welcome
		indices := birthdateRx.FindStringSubmatchIndex(v)
		if indices == nil {
			return errors.New(`invalid pattern for birthdate`)
		}
		var tmp BirthdateClaim

		year, err := strconv.ParseInt(v[indices[2]:indices[3]], 10, 64)
		if err != nil {
			return errors.New(`failed to parse birthdate year`)
		}
		if year > 0 {
			var v int = int(year)
			tmp.year = &v
		}

		month, err := strconv.ParseInt(v[indices[4]:indices[5]], 10, 64)
		if err != nil {
			return errors.New(`failed to parse birthdate month`)
		}
		if month > 0 {
			var v int = int(month)
			tmp.month = &v
		}

		day, err := strconv.ParseInt(v[indices[6]:indices[7]], 10, 64)
		if err != nil {
			return errors.New(`failed to parse birthdate day`)
		}
		if day > 0 {
			var v int = int(day)
			tmp.day = &v
		}

		*b = tmp
		return nil
	default:
		return errors.Errorf(`invalid type for birthdate: %T`, v)
	}
}

func (b BirthdateClaim) encode(dst io.Writer) {
	fmt.Fprintf(dst, "%d-%02d-%02d", b.Year(), b.Month(), b.Day())
}

func (b BirthdateClaim) String() string {
	var buf bytes.Buffer
	b.encode(&buf)
	return buf.String()
}

func (b BirthdateClaim) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	b.encode(&buf)
	return buf.Bytes(), nil
}
