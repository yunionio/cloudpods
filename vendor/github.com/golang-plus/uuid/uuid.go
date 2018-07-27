package uuid

import (
	"bytes"
	"fmt"

	"github.com/golang-plus/uuid/internal"
	"github.com/golang-plus/uuid/internal/dcesecurity"
	"github.com/golang-plus/uuid/internal/namebased/md5"
	"github.com/golang-plus/uuid/internal/namebased/sha1"
	"github.com/golang-plus/uuid/internal/random"
	"github.com/golang-plus/uuid/internal/timebased"
)

// UUID respresents an UUID type compliant with specification in RFC 4122.
type UUID [16]byte

// Layout returns layout of UUID.
func (u UUID) Layout() Layout {
	return Layout(internal.GetLayout(u[:]))
}

// Version returns version of UUID.
func (u UUID) Version() Version {
	return Version(internal.GetVersion(u[:]))
}

// Equal returns true if current uuid equal to passed uuid.
func (u UUID) Equal(another UUID) bool {
	return bytes.EqualFold(u[:], another[:])
}

// Format returns the formatted string of UUID.
func (u UUID) Format(style Style) string {
	switch style {
	case StyleWithoutDash:
		return fmt.Sprintf("%x", u[:])
	//case StyleStandard:
	default:
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", u[:4], u[4:6], u[6:8], u[8:10], u[10:])
	}
}

// String returns the string of UUID with standard style(xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx | 8-4-4-4-12).
func (u UUID) String() string {
	return u.Format(StyleStandard)
}

var (
	// Nil represents the Nil UUID (00000000-0000-0000-0000-000000000000).
	Nil = UUID{}
)

// NewTimeBased returns a new time based UUID (version 1).
func NewTimeBased() (UUID, error) {
	u, err := timebased.NewUUID()
	if err != nil {
		return Nil, err
	}

	uuid := UUID{}
	copy(uuid[:], u)

	return uuid, nil
}

// NewV1 is short of NewTimeBased.
func NewV1() (UUID, error) {
	return NewTimeBased()
}

// NewDCESecurity returns a new DCE security UUID (version 2).
func NewDCESecurity(domain Domain) (UUID, error) {
	u, err := dcesecurity.NewUUID(dcesecurity.Domain(domain))
	if err != nil {
		return Nil, err
	}

	uuid := UUID{}
	copy(uuid[:], u)

	return uuid, nil
}

// NewV2 is short of NewDCESecurity.
func NewV2(domain Domain) (UUID, error) {
	return NewDCESecurity(domain)
}

// NewNameBasedMD5 returns a new name based UUID with MD5 hash (version 3).
func NewNameBasedMD5(namespace, name string) (UUID, error) {
	u, err := md5.NewUUID(namespace, name)
	if err != nil {
		return Nil, err
	}

	uuid := UUID{}
	copy(uuid[:], u)

	return uuid, nil
}

// NewV3 is short of NewNameBasedMD5.
func NewV3(namespace, name string) (UUID, error) {
	return NewNameBasedMD5(namespace, name)
}

// NewRandom returns a new random UUID (version 4).
func NewRandom() (UUID, error) {
	u, err := random.NewUUID()
	if err != nil {
		return Nil, err
	}

	uuid := UUID{}
	copy(uuid[:], u)

	return uuid, nil
}

// NewV4 is short of NewRandom.
func NewV4() (UUID, error) {
	return NewRandom()
}

// New is short of NewRandom.
func New() (UUID, error) {
	return NewRandom()
}

// NewNameBasedSHA1 returns a new name based UUID with SHA1 hash (version 5).
func NewNameBasedSHA1(namespace, name string) (UUID, error) {
	u, err := sha1.NewUUID(namespace, name)
	if err != nil {
		return Nil, err
	}

	uuid := UUID{}
	copy(uuid[:], u)

	return uuid, nil
}

// NewV5 is short of NewNameBasedSHA1.
func NewV5(namespace, name string) (UUID, error) {
	return NewNameBasedSHA1(namespace, name)
}
