package timebased

import (
	"crypto/rand"
	"encoding/binary"
	"net"
	"sync"
	"time"

	"github.com/golang-plus/uuid/internal"

	"github.com/golang-plus/errors"
)

const (
	// Intervals bewteen 1/1/1970 and 15/10/1582 (Julain days of 1 Jan 1970 - Julain days of 15 Oct 1582) * 100-Nanoseconds Per Day
	intervals = (2440587 - 2299160) * 86400 * 10000000
)

var (
	lastGenerated time.Time  // last generated time
	clockSequence uint16     // clock sequence for same tick
	nodeID        []byte     // node id (MAC Address)
	locker        sync.Mutex // global lock
)

// NewUUID returns a new time-based uuid.
func NewUUID() ([]byte, error) {
	// Get and release a global lock
	locker.Lock()
	defer locker.Unlock()

	uuid := make([]byte, 16)

	// get timestamp
	now := time.Now().UTC()
	timestamp := uint64(now.UnixNano()/100) + intervals // get timestamp
	if !now.After(lastGenerated) {
		clockSequence++ // last generated time known, then just increment clock sequence
	} else {
		b := make([]byte, 2)
		_, err := rand.Read(b)
		if err != nil {
			return nil, errors.Wrap(err, "could not generate clock sequence")
		}
		clockSequence = uint16(int(b[0])<<8 | int(b[1])) // set to a random value (network byte order)
	}

	lastGenerated = now // remember the last generated time

	timeLow := uint32(timestamp & 0xffffffff)
	timeMiddle := uint16((timestamp >> 32) & 0xffff)
	timeHigh := uint16((timestamp >> 48) & 0xfff)

	// network byte order(BigEndian)
	binary.BigEndian.PutUint32(uuid[0:], timeLow)
	binary.BigEndian.PutUint16(uuid[4:], timeMiddle)
	binary.BigEndian.PutUint16(uuid[6:], timeHigh)
	binary.BigEndian.PutUint16(uuid[8:], clockSequence)

	// get node id(mac address)
	if nodeID == nil {
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil, errors.Wrap(err, "could not get network interfaces")
		}

		for _, i := range interfaces {
			if len(i.HardwareAddr) >= 6 {
				nodeID = make([]byte, 6)
				copy(nodeID, i.HardwareAddr)
				break
			}
		}

		if nodeID == nil {
			nodeID = make([]byte, 6)
			_, err := rand.Read(nodeID)
			if err != nil {
				return nil, errors.Wrap(err, "could not generate node id")
			}
		}
	}

	copy(uuid[10:], nodeID)

	// set version(v1)
	internal.SetVersion(uuid, internal.VersionTimeBased)
	// set layout(RFC4122)
	internal.SetLayout(uuid, internal.LayoutRFC4122)

	return uuid, nil
}
