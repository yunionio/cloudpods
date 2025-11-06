package envpprof

import (
	"expvar"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/anacrolix/log"
)

var (
	pprofDir = filepath.Join(os.Getenv("HOME"), "pprof")
	heap     bool
)

func writeHeapProfile() {
	os.Mkdir(pprofDir, 0750)
	f, err := ioutil.TempFile(pprofDir, "heap")
	if err != nil {
		log.Printf("error creating heap profile file: %s", err)
		return
	}
	defer f.Close()
	pprof.WriteHeapProfile(f)
	log.Printf("wrote heap profile to %q", f.Name())
}

// Stop ends CPU profiling, waiting for writes to complete. If heap profiling is enabled, it also
// writes the heap profile to a file. Stop should be deferred from main if cpu or heap profiling
// are to be used through envpprof.
func Stop() {
	// Should we check if CPU profiling was initiated through this package?
	pprof.StopCPUProfile()
	trace.Stop()
	if heap {
		// Can or should we do this concurrently with stopping CPU profiling?
		writeHeapProfile()
	}
}

func startHTTP(value string) {
	var l net.Listener
	if value == "" {
		for port := uint16(6061); port != 6060; port++ {
			var err error
			l, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			if err == nil {
				break
			}
		}
		if l == nil {
			log.Print("unable to create envpprof listener for http")
			return
		}
	} else {
		var addr string
		_, _, err := net.SplitHostPort(value)
		if err == nil {
			addr = value
		} else {
			addr = "localhost:" + value
		}
		l, err = net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("(pid=%d) envpprof serving http://%s", os.Getpid(), l.Addr())
	go func() {
		defer l.Close()
		log.Printf("error serving http on envpprof listener: %s", http.Serve(l, nil))
	}()
}

func init() {
	expvar.Publish("numGoroutine", expvar.Func(func() interface{} { return runtime.NumGoroutine() }))
	_var := os.Getenv("GOPPROF")
	if _var == "" {
		return
	}
	for _, item := range strings.Split(os.Getenv("GOPPROF"), ",") {
		equalsPos := strings.IndexByte(item, '=')
		var key, value string
		if equalsPos < 0 {
			key = item
		} else {
			key = item[:equalsPos]
			value = item[equalsPos+1:]
		}
		switch key {
		case "http":
			startHTTP(value)
		case "cpu":
			os.Mkdir(pprofDir, 0750)
			f, err := ioutil.TempFile(pprofDir, "cpu")
			if err != nil {
				log.Printf("error creating cpu pprof file: %s", err)
				break
			}
			err = pprof.StartCPUProfile(f)
			if err != nil {
				log.Printf("error starting cpu profiling: %s", err)
				break
			}
			log.Printf("cpu profiling to file %q", f.Name())
		case "block":
			// Taken from Safe Rate at
			// https://github.com/DataDog/go-profiler-notes/blob/main/guide/README.md#go-profilers.
			runtime.SetBlockProfileRate(10000)
		case "heap":
			heap = true
		case "mutex":
			// Taken from Safe Rate at
			// https://github.com/DataDog/go-profiler-notes/blob/main/guide/README.md#go-profilers.
			runtime.SetMutexProfileFraction(100)
		case "trace":
			f, err := ioutil.TempFile(pprofDir, "trace")
			if err != nil {
				log.Printf("error creating trace file: %v", err)
				break
			}
			err = trace.Start(f)
			if err != nil {
				log.Printf("error starting tracing: %v", err)
				break
			}
			log.Printf("tracing to file %q", f.Name())
		default:
			log.Printf("unexpected GOPPROF key %q", key)
		}
	}
}
