package jbbsreader

import (
	"bufio"
	"code.google.com/p/go.text/encoding/japanese"
	"code.google.com/p/go.text/transform"
	"io"
	"log"
	"net/http"
	"time"
)

// The global rate per minute limit for requests to JBBS.
var (
	ratePerMinuteLimit = 3
	burst              = 3
)

func init() {
	SetGlobalRateLimitRPM(ratePerMinuteLimit, burst)
}

var rpmThrottle chan bool

// datEncoding specifies the encoding the JBBS response is in.
var datEncoding = japanese.EUCJP

// SetGlobalRateLimitRPM sets JBBS requests rate per limit, allowing some bursts.
func SetGlobalRateLimitRPM(rpm int, burst int) {
	rpmThrottle = make(chan bool, burst)
	for i := 0; i < burst; i++ {
		rpmThrottle <- true
	}

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

	log.Printf("GET %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ls, err := lines(resp.Body)
	if err != nil {
		return nil, err
	}

	return ls, nil
}

// lines reads each non-empty line from r, with no EOL markers such as CR, LF, and CR+LF.
func lines(r io.Reader) (ls []string, _ error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		l := scanner.Bytes()
		if len(l) == 0 {
			continue
		}

		bs, n, err := transform.Bytes(datEncoding.NewDecoder(), l)
		if err != nil {
			log.Printf("Could not convert EUC string %q to UTF-8, position %v: %v. Ignored.", string(l), n, err)
			continue
		}

		ls = append(ls, string(bs))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ls, nil
}
