package downloader

import (
	"compress/zlib"
	"io"
	"net/http"
	"os"
	"time"

	"yunion.io/x/log"
)

const (
	CHUNK_SIZE         = 1024 * 8
	DEFAULT_RATE_LIMIT = 50
	COMPRESS_LEVEL     = 1
)

type SDownloadProvider struct {
	w         http.ResponseWriter
	rateLimit int
	compress  bool
}

func NewDownloadProvider(w http.ResponseWriter, compress bool, rateLimit int) *SDownloadProvider {
	if rateLimit <= 0 {
		rateLimit = DEFAULT_RATE_LIMIT
	}
	return &SDownloadProvider{w, rateLimit, compress}
}

func (d *SDownloadProvider) Start(
	prepareDownload func() error, onDownloadComplete func(),
	downloadFilePath string, headers http.Header,
) error {
	if prepareDownload != nil {
		if err := prepareDownload(); err != nil {
			log.Errorln(err)
			return err
		}
	}
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/octet-stream")
	}
	for k := range headers {
		d.w.Header().Add(k, headers.Get(k))
	}

	fi, err := os.Open(downloadFilePath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	defer fi.Close()

	var (
		end                 = false
		chunk               = make([]byte, CHUNK_SIZE)
		writer    io.Writer = d.w
		startTime           = time.Now()
		sendBytes           = 0
	)

	if d.compress {
		zw, err := zlib.NewWriterLevel(d.w, COMPRESS_LEVEL)
		if err != nil {
			log.Errorln(err)
			return err
		}
		writer = zw
		defer zw.Flush() // it's cool
		defer zw.Close()
	}

	for !end {
		if _, err := fi.Read(chunk); err == io.EOF {
			end = true
		} else if err != nil && err != io.EOF {
			log.Errorln(err)
			return err
		}

		if size, err := writer.Write(chunk); err != nil {
			log.Errorln(err)
			return err
		} else {
			sendBytes += size
			timeDur := time.Now().Sub(startTime)
			exceptDur := float64(sendBytes) / 1000.0 / 1000.0 / float64(d.rateLimit)
			if exceptDur > timeDur.Seconds() {
				time.Sleep(time.Duration(exceptDur-timeDur.Seconds()) * time.Second)
			}
		}
	}

	// if d.compress {
	// 	zw := writer.(*zlib.Writer)
	// 	zw.Flush()
	// }

	sendMb := float64(sendBytes) / 1000.0 / 1000.0
	timeDur := time.Now().Sub(startTime)
	log.Infof("Send data: %fMB rate: %fMB/sec", sendMb/timeDur.Seconds())

	if onDownloadComplete != nil {
		onDownloadComplete()
	}
	return nil
}
