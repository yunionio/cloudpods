package uuid

import (
	"github.com/golang-plus/uuid/internal"
)

// Version represents the version of UUID. See page 7 in RFC 4122.
type Version byte

// Versions.
const (
	// VersionUnknown represents unknown version.
	VersionUnknown = Version(internal.VersionUnknown)
	// VersionTimeBased represents the time-based version (Version 1).
	VersionTimeBased = Version(internal.VersionTimeBased)
	// VersionDCESecurity represents the DCE security version, with embedded POSIX UIDs (Version 2).
	VersionDCESecurity = Version(internal.VersionDCESecurity)
	// VersionNameBasedMD5 represents the name-based version that uses MD5 hashing (Version 3).
	VersionNameBasedMD5 = Version(internal.VersionNameBasedMD5)
	// VersionRandom represents the randomly or pseudo-randomly generated version (Version 4).
	VersionRandom = Version(internal.VersionRandom)
	// VersionNameBasedSHA1 represents the name-based version that uses SHA-1 hashing (Version 5).
	VersionNameBasedSHA1 = Version(internal.VersionNameBasedSHA1)
)

// Short names of versions.
const (
	V1 = VersionTimeBased
	V2 = VersionDCESecurity
	V3 = VersionNameBasedMD5
	V4 = VersionRandom
	V5 = VersionNameBasedSHA1
)

// String returns English description of Version.
func (v Version) String() string {
	switch v {
	case VersionTimeBased:
		return "Version 1: Time-Based"
	case VersionDCESecurity:
		return "Version 2: DCE Security With Embedded POSIX UIDs"
	case VersionNameBasedMD5:
		return "Version 3: Name-Based (MD5)"
	case VersionRandom:
		return "Version 4: Randomly OR Pseudo-Randomly Generated"
	case VersionNameBasedSHA1:
		return "Version 5: Name-Based (SHA-1)"
	default:
		return "Version: Unknwon"
	}
}
