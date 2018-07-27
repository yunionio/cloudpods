package internal

// Version represents the version of UUID. See page 7 in RFC 4122.
type Version byte

// Version List
const (
	VersionUnknown       Version = iota // Unknwon
	VersionTimeBased                    // V1: The time-based version
	VersionDCESecurity                  // V2: The DCE security version, with embedded POSIX UIDs
	VersionNameBasedMD5                 // V3: The name-based version that uses MD5 hashing
	VersionRandom                       // V4: The randomly or pseudo-randomly generated version
	VersionNameBasedSHA1                // V5: The name-based version that uses SHA-1 hashing
)

// SetVersion sets the version for uuid.
// This is intended to be called from the New function in packages that implement uuid generating functions.
func SetVersion(uuid []byte, version Version) {
	switch version {
	case VersionTimeBased:
		uuid[6] = (uuid[6] | 0x10) & 0x1f
	case VersionDCESecurity:
		uuid[6] = (uuid[6] | 0x20) & 0x2f
	case VersionNameBasedMD5:
		uuid[6] = (uuid[6] | 0x30) & 0x3f
	case VersionRandom:
		uuid[6] = (uuid[6] | 0x40) & 0x4f
	case VersionNameBasedSHA1:
		uuid[6] = (uuid[6] | 0x50) & 0x5f
	default:
		panic("version is unknown")
	}
}

// GetVersion gets the version of uuid.
func GetVersion(uuid []byte) Version {
	ver := uuid[6] >> 4
	if ver > 0 && ver < 6 {
		return Version(ver)
	}

	return VersionUnknown
}
