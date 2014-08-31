package jbbsreader

import (
	"bufio"
	"code.google.com/p/go.text/encoding/japanese"
	"code.google.com/p/go.text/transform"
	"io"
	"net/http"
	"time"
)

// The global rate per minute limit for requests to JBBS.
var ratePerMinuteLimit = 3

func init() {
	SetGlobalRateLimitRPM(ratePerMinuteLimit)
}

var rpmThrottle chan bool

// datEncoding specifies the encoding the JBBS response is in.
var datEncoding = japanese.EUCJP

// SetGlobalRateLimitRPM sets JBBS requests rate per limit.
func SetGlobalRateLimitRPM(rpm int) {
	rpmThrottle = make(chan bool)
	go func() {
		for _ = range time.Tick(time.Minute / time.Duration(rpm)) {
			select {
			case rpmThrottle <- true:
			default:
			}
		}
	}()
}

// getLines sends a HTTP GET request to the url, and returns a slice of lines of the response.
// This is a var so to stub it in tests.
var getLines = func(url string) ([]string, error) {
	<-rpmThrottle
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and transform to UTF-8 encoding.
	r := transform.NewReader(resp.Body, datEncoding.NewDecoder())
	ls, err := lines(r)
	if err != nil {
		return nil, err
	}

	return ls, nil
}

// lines reads each non-empty line from r, with no EOL markers such as CR, LF, and CR+LF.
func lines(r io.Reader) (ls []string, _ error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		l := scanner.Text()
		if len(l) > 0 {
			ls = append(ls, l)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ls, nil
}
