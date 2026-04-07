// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:generate msgp -unexported -marshal=false -o=stats_msgp.go -tests=false

package tracer

import (
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/DataDog/sketches-go/ddsketch"
	"google.golang.org/protobuf/proto"
)

// aggregableSpan holds necessary information about a span that can be used to
// aggregate statistics in a bucket.
type aggregableSpan struct {
	// key specifies the aggregation key under which this span can be placed into
	// grouped inside a bucket.
	key aggregation

	Start, Duration int64
	Error           int32
	TopLevel        bool
}

// defaultStatsBucketSize specifies the default span of time that will be
// covered in one stats bucket.
var defaultStatsBucketSize = (10 * time.Second).Nanoseconds()

// concentrator aggregates and stores statistics on incoming spans in time buckets,
// flushing them occasionally to the underlying transport located in the given
// tracer config.
type concentrator struct {
	// In specifies the channel to be used for feeding data to the concentrator.
	// In order for In to have a consumer, the concentrator must be started using
	// a call to Start.
	In chan *aggregableSpan

	// mu guards below fields
	mu sync.Mutex

	// buckets maintains a set of buckets, where the map key represents
	// the starting point in time of that bucket, in nanoseconds.
	buckets map[int64]*rawBucket

	// stopped reports whether the concentrator is stopped (when non-zero)
	stopped uint32

	wg           sync.WaitGroup // waits for any active goroutines
	bucketSize   int64          // the size of a bucket in nanoseconds
	stop         chan struct{}  // closing this channel triggers shutdown
	cfg          *config        // tracer startup configuration
	statsdClient statsdClient   // statsd client for sending metrics.
}

// newConcentrator creates a new concentrator using the given tracer
// configuration c. It creates buckets of bucketSize nanoseconds duration.
func newConcentrator(c *config, bucketSize int64) *concentrator {
	return &concentrator{
		In:         make(chan *aggregableSpan, 10000),
		bucketSize: bucketSize,
		stopped:    1,
		buckets:    make(map[int64]*rawBucket),
		cfg:        c,
	}
}

// alignTs returns the provided timestamp truncated to the bucket size.
// It gives us the start time of the time bucket in which such timestamp falls.
func alignTs(ts, bucketSize int64) int64 { return ts - ts%bucketSize }

// Start starts the concentrator. A started concentrator needs to be stopped
// in order to gracefully shut down, using Stop.
func (c *concentrator) Start() {
	if atomic.SwapUint32(&c.stopped, 0) == 0 {
		// already running
		log.Warn("(*concentrator).Start called more than once. This is likely a programming error.")
		return
	}
	c.stop = make(chan struct{})
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		tick := time.NewTicker(time.Duration(c.bucketSize) * time.Nanosecond)
		defer tick.Stop()
		c.runFlusher(tick.C)
	}()
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runIngester()
	}()
}

// runFlusher runs the flushing loop which sends stats to the underlying transport.
func (c *concentrator) runFlusher(tick <-chan time.Time) {
	for {
		select {
		case now := <-tick:
			c.flushAndSend(now, withoutCurrentBucket)
		case <-c.stop:
			return
		}
	}
}

// statsd returns any tracer configured statsd client, or a no-op.
func (c *concentrator) statsd() statsdClient {
	if c.statsdClient == nil {
		return &statsd.NoOpClient{}
	}
	return c.statsdClient
}

// runIngester runs the loop which accepts incoming data on the concentrator's In
// channel.
func (c *concentrator) runIngester() {
	for {
		select {
		case s := <-c.In:
			c.statsd().Incr("datadog.tracer.stats.spans_in", nil, 1)
			c.add(s)
		case <-c.stop:
			return
		}
	}
}

// add adds s into the concentrator's internal stats buckets.
func (c *concentrator) add(s *aggregableSpan) {
	c.mu.Lock()
	defer c.mu.Unlock()

	btime := alignTs(s.Start+s.Duration, c.bucketSize)
	b, ok := c.buckets[btime]
	if !ok {
		b = newRawBucket(uint64(btime), c.bucketSize)
		c.buckets[btime] = b
	}
	b.handleSpan(s)
}

// Stop stops the concentrator and blocks until the operation completes.
func (c *concentrator) Stop() {
	if atomic.SwapUint32(&c.stopped, 1) > 0 {
		return
	}
	close(c.stop)
	c.wg.Wait()
drain:
	for {
		select {
		case s := <-c.In:
			c.statsd().Incr("datadog.tracer.stats.spans_in", nil, 1)
			c.add(s)
		default:
			break drain
		}
	}
	c.flushAndSend(time.Now(), withCurrentBucket)
}

const (
	withCurrentBucket    = true
	withoutCurrentBucket = false
)

// flushAndSend flushes all the stats buckets with the given timestamp and sends them using the transport specified in
// the concentrator config. The current bucket is only included if includeCurrent is true, such as during shutdown.
func (c *concentrator) flushAndSend(timenow time.Time, includeCurrent bool) {
	sp := func() statsPayload {
		c.mu.Lock()
		defer c.mu.Unlock()
		now := timenow.UnixNano()
		sp := statsPayload{
			Hostname: c.cfg.hostname,
			Env:      c.cfg.env,
			Version:  c.cfg.version,
			Stats:    make([]statsBucket, 0, len(c.buckets)),
		}
		for ts, srb := range c.buckets {
			if !includeCurrent && ts > now-c.bucketSize {
				// do not flush the current bucket
				continue
			}
			log.Debug("Flushing bucket %d", ts)
			sp.Stats = append(sp.Stats, srb.Export())
			delete(c.buckets, ts)
		}
		return sp
	}()

	if len(sp.Stats) == 0 {
		// nothing to flush
		return
	}
	c.statsd().Incr("datadog.tracer.stats.flush_payloads", nil, 1)
	c.statsd().Incr("datadog.tracer.stats.flush_buckets", nil, float64(len(sp.Stats)))
	if err := c.cfg.transport.sendStats(&sp); err != nil {
		c.statsd().Incr("datadog.tracer.stats.flush_errors", nil, 1)
		log.Error("Error sending stats payload: %v", err)
	}
}

// aggregation specifies a uniquely identifiable key under which a certain set
// of stats are grouped inside a bucket.
type aggregation struct {
	Name       string
	Type       string
	Resource   string
	Service    string
	StatusCode uint32
	Synthetics bool
}

type rawBucket struct {
	start, duration uint64
	data            map[aggregation]*rawGroupedStats
}

func newRawBucket(btime uint64, bsize int64) *rawBucket {
	return &rawBucket{
		start:    btime,
		duration: uint64(bsize),
		data:     make(map[aggregation]*rawGroupedStats),
	}
}

func (sb *rawBucket) handleSpan(s *aggregableSpan) {
	gs, ok := sb.data[s.key]
	if !ok {
		gs = newRawGroupedStats()
		sb.data[s.key] = gs
	}
	if s.TopLevel {
		gs.topLevelHits++
	}
	gs.hits++
	if s.Error != 0 {
		gs.errors++
	}
	gs.duration += uint64(s.Duration)
	// alter resolution of duration distro
	trundur := nsTimestampToFloat(s.Duration)
	if s.Error != 0 {
		gs.errDistribution.Add(trundur)
	} else {
		gs.okDistribution.Add(trundur)
	}
}

// Export transforms a RawBucket into a statsBucket, typically used
// before communicating data to the API, as RawBucket is the internal
// type while statsBucket is the public, shared one.
func (sb *rawBucket) Export() statsBucket {
	csb := statsBucket{
		Start:    sb.start,
		Duration: sb.duration,
		Stats:    make([]groupedStats, len(sb.data)),
	}
	for k, v := range sb.data {
		b, err := v.export(k)
		if err != nil {
			log.Error("Could not export stats bucket: %v.", err)
			continue
		}
		csb.Stats = append(csb.Stats, b)
	}
	return csb
}

type rawGroupedStats struct {
	hits            uint64
	topLevelHits    uint64
	errors          uint64
	duration        uint64
	okDistribution  *ddsketch.DDSketch
	errDistribution *ddsketch.DDSketch
}

func newRawGroupedStats() *rawGroupedStats {
	const (
		// relativeAccuracy is the value accuracy we have on the percentiles. For example, we can
		// say that p99 is 100ms +- 1ms
		relativeAccuracy = 0.01
		// maxNumBins is the maximum number of bins of the ddSketch we use to store percentiles.
		// It can affect relative accuracy, but in practice, 2048 bins is enough to have 1% relative accuracy from
		// 80 micro second to 1 year: http://www.vldb.org/pvldb/vol12/p2195-masson.pdf
		maxNumBins = 2048
	)
	okSketch, err := ddsketch.LogCollapsingLowestDenseDDSketch(relativeAccuracy, maxNumBins)
	if err != nil {
		log.Error("Error when creating ddsketch: %v", err)
	}
	errSketch, err := ddsketch.LogCollapsingLowestDenseDDSketch(relativeAccuracy, maxNumBins)
	if err != nil {
		log.Error("Error when creating ddsketch: %v", err)
	}
	return &rawGroupedStats{
		okDistribution:  okSketch,
		errDistribution: errSketch,
	}
}

func (s *rawGroupedStats) export(k aggregation) (groupedStats, error) {
	msg := s.okDistribution.ToProto()
	okSummary, err := proto.Marshal(msg)
	if err != nil {
		return groupedStats{}, err
	}
	msg = s.errDistribution.ToProto()
	errSummary, err := proto.Marshal(msg)
	if err != nil {
		return groupedStats{}, err
	}
	return groupedStats{
		Service:        k.Service,
		Name:           k.Name,
		Resource:       k.Resource,
		HTTPStatusCode: k.StatusCode,
		Type:           k.Type,
		Hits:           s.hits,
		Errors:         s.errors,
		Duration:       s.duration,
		TopLevelHits:   s.topLevelHits,
		OkSummary:      okSummary,
		ErrorSummary:   errSummary,
		Synthetics:     k.Synthetics,
	}, nil
}

// nsTimestampToFloat converts a nanosec timestamp into a float nanosecond timestamp truncated to a fixed precision
func nsTimestampToFloat(ns int64) float64 {
	// 10 bits precision (any value will be +/- 1/1024)
	const roundMask int64 = 1 << 10
	var shift uint
	for ns > roundMask {
		ns = ns >> 1
		shift++
	}
	return float64(ns << shift)
}
