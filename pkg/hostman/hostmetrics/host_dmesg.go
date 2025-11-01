package hostmetrics

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	kmsgPath      = "/dev/kmsg"
	batchSize     = 100
	flushInterval = 100 * time.Second
)

type SHostDmesgCollector struct {
	host IHostInfo

	mu       sync.Mutex
	buffer   []compute.SKmsgEntry
	bootTime time.Time
}

func NewHostDmesgCollector(hostInfo IHostInfo) *SHostDmesgCollector {
	return &SHostDmesgCollector{
		host:   hostInfo,
		mu:     sync.Mutex{},
		buffer: make([]compute.SKmsgEntry, 0),
	}
}

// /dev/kmsg
// level;sequence,timestamp[us];message
// include/linux/kern_levels.h
// #define LOGLEVEL_EMERG		0	/* system is unusable */
// #define LOGLEVEL_ALERT		1	/* action must be taken immediately */
// #define LOGLEVEL_CRIT		2	/* critical conditions */
// #define LOGLEVEL_ERR		3	/* error conditions */
// #define LOGLEVEL_WARNING	4	/* warning conditions */
// #define LOGLEVEL_NOTICE		5	/* normal but significant condition */
// #define LOGLEVEL_INFO		6	/* informational */
// #define LOGLEVEL_DEBUG		7	/* debug-level messages */

func (c *SHostDmesgCollector) Start() {
	f, err := os.Open(kmsgPath)
	if err != nil {
		log.Errorf("failed open %s: %s", kmsgPath, err)
		return
	}
	defer f.Close()

	bootTime, err := getBootTime()
	if err != nil {
		log.Errorf("failed get boot time %s", err)
		return
	}
	c.bootTime = bootTime

	var currentBootStamp = bootTime.Unix()
	var lastSeq = 0

	readerState, err := c.loadState()
	if err != nil {
		log.Errorf("failed load readers state %s", err)
	} else if readerState != nil {
		if readerState.BootStamp == currentBootStamp {
			lastSeq = readerState.LastSeq
		}
	}
	log.Infof("Start dmesg reader from seq %d", lastSeq)

	go func() {
		for range time.Tick(flushInterval) {
			c.mu.Lock()
			c.flushBuffer()
			c.mu.Unlock()
		}
	}()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		entry, err := c.parseKmsgLine(line, bootTime)
		if err != nil {
			log.Errorf("failed parse kmsg line %s: %s", line, err)
			continue
		}
		if entry.Seq <= lastSeq {
			continue
		}
		// 只上传 warn 以上级别的日志
		if entry.Level > 4 || c.isNoise(entry) {
			continue
		}

		c.mu.Lock()
		c.buffer = append(c.buffer, *entry)
		if len(c.buffer) >= batchSize {
			c.flushBuffer()
		}
		c.mu.Unlock()
	}
}

func (c *SHostDmesgCollector) isNoise(entry *compute.SKmsgEntry) bool {
	if strings.HasPrefix(entry.Message, "IPVS:") {
		return true
	}
	return false
}

// flush buffer util success
func (c *SHostDmesgCollector) flushBuffer() {
	if len(c.buffer) == 0 {
		return
	}
	seq := c.buffer[len(c.buffer)-1].Seq

	for {
		err := c.host.ReportHostDmesg(c.buffer)
		if err != nil {
			log.Errorf("failed report host dmesg %s", err)
			time.Sleep(time.Second * 30)
			continue
		}
		break
	}

	if err := c.saveState(seq); err != nil {
		log.Errorf("failed save dmesg reader state: %s", err)
	}

	c.buffer = c.buffer[:0]
}

func (c *SHostDmesgCollector) loadState() (*ReaderState, error) {
	dmesgStatePath := path.Join(filepath.Dir(options.HostOptions.ServersPath), "dmesg_reader_state")
	if !fileutils2.Exists(dmesgStatePath) {
		return nil, nil
	}
	data, err := fileutils2.FileGetContents(dmesgStatePath)
	if err != nil {
		return nil, err
	}
	jdata, err := jsonutils.ParseString(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed parse dmesg reader state")
	}
	var s ReaderState
	err = jdata.Unmarshal(&s)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshal reader state")
	}
	return &s, nil
}

func (c *SHostDmesgCollector) saveState(seq int) error {
	state := &ReaderState{
		LastSeq:   seq,
		BootStamp: c.bootTime.Unix(),
	}
	jstate := jsonutils.Marshal(state)
	dmesgStatePath := path.Join(filepath.Dir(options.HostOptions.ServersPath), "dmesg_reader_state")
	return fileutils2.FilePutContents(dmesgStatePath, jstate.String(), false)
}

type ReaderState struct {
	LastSeq   int   `json:"last_seq"`
	BootStamp int64 `json:"boot_stamp"` // UNIX seconds of boot time
}

func getBootTime() (time.Time, error) {
	data, err := fileutils2.FileGetContents("/proc/uptime")
	if err != nil {
		return time.Time{}, err
	}
	fields := strings.Fields(data)
	if len(fields) < 1 {
		return time.Time{}, fmt.Errorf("invalid /proc/uptime")
	}
	uptimeSec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(-time.Duration(uptimeSec * float64(time.Second))), nil
}

func (c *SHostDmesgCollector) parseKmsgLine(line string, bootTime time.Time) (*compute.SKmsgEntry, error) {
	parts := strings.SplitN(line, ";", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid kmsg line: %s", line)
	}

	meta := strings.Split(parts[0], ",")
	if len(meta) < 3 {
		return nil, fmt.Errorf("invalid meta: %s", parts[0])
	}

	levelStr := strings.Trim(meta[0], "<>")
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		return nil, err
	}

	seq, _ := strconv.Atoi(meta[1])
	timestamp, _ := strconv.ParseUint(meta[2], 10, 64)
	rel := time.Duration(timestamp) * time.Microsecond
	abs := bootTime.Add(rel)

	return &compute.SKmsgEntry{
		Level:   level,
		Seq:     seq,
		Message: parts[1],
		Time:    abs,
	}, nil
}
