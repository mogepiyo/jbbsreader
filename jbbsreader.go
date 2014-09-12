// Package jbbsreader provides functionality for reading from a specified JBBS board.
package jbbsreader

import (
	"fmt"
	"regexp"
	"strconv"
)

// The URL formats to send requests to.
const (
	jbbsURLFormat        = "http://jbbs.shitaraba.net/%s/%s/"
	jbbsSubjectURLFormat = "http://jbbs.shitaraba.net/%s/%s/subject.txt"
	jbbsDatURLFormat     = "http://jbbs.shitaraba.net/bbs/rawmode.cgi/%s/%s/%s"
)

// The regexps to parse dat from JBBS.
const (
	subjectPattern = `^([[:digit:]]*)\.cgi,(.*)\(([[:digit:]]*)\)$`
	datPattern     = `^([[:digit:]]*)<>(.*)<>(.*)<>(.*)<>(.*)<>(.*)<>(.*)$`
)

var (
	subjectRegexp = regexp.MustCompile(subjectPattern)
	datRegexp     = regexp.MustCompile(datPattern)
)

// Board represents a JBBS board whose URL is 'http://jbbs.shitaraba.net/[category]/[id]/'.
type Board struct {
	URL          string
	Category, ID string
}

// NewBoard constructs a Board specified by category and id.
func NewBoard(category, id string) *Board {
	return &Board{
		URL:      fmt.Sprintf(jbbsURLFormat, category, id),
		Category: category,
		ID:       id,
	}
}

// Threads returns the list of threads in the board.
// b.Category and b.ID must be set.
func (b *Board) Threads() (threads []*Thread, _ error) {
	subjectURL := fmt.Sprintf(jbbsSubjectURLFormat, b.Category, b.ID)

	// Read from JBBS.
	lines, err := getLines(subjectURL)
	if err != nil {
		return nil, err
	}

	// Parse each line.
	for i, line := range lines {
		t := newThread(b, line)
		if t == nil {
			return nil, fmt.Errorf("could not parse subject.txt:%v %q", i, line)
		}
		threads = append(threads, t)
	}

	return threads, nil
}

// FeedNewResponses continuously feeds new responses in all threads of the board to the chan.
// b.Category and b.ID must be set.
func (b *Board) FeedNewResponses() (<-chan *Response, <-chan error) {
	resp, errc := make(chan *Response), make(chan error, 1)

	go func() {
		defer func() {
			close(resp)
			close(errc)
		}()

		known := make(map[string]bool)

		// Load initial known responses.
		respChan, errChan := b.FeedResponsesOnce()
		for r := range respChan {
			known[key(r)] = true
		}
		if err := <-errChan; err != nil {
			errc <- err
			return
		}

		// Poll JBBS and emit unknown responses.
		for {
			respChan, errChan = b.FeedResponsesOnce()
			for r := range respChan {
				if !known[key(r)] {
					known[key(r)] = true
					resp <- r
				}
			}
			if err := <-errChan; err != nil {
				errc <- err
				return
			}
		}
	}()

	return resp, errc
}

// FeedResponsesOnce reads all responses in all threads of the board to the chan.
// b.Category and b.ID must be set.
func (b *Board) FeedResponsesOnce() (<-chan *Response, <-chan error) {
	resp, errc := make(chan *Response), make(chan error, 1)

	go func() {
		defer func() {
			close(resp)
			close(errc)
		}()

		threads, err := b.Threads()
		if err != nil {
			errc <- err
			return
		}

		for _, t := range threads {
			responses, err := t.Responses()
			if err != nil {
				errc <- err
				return
			}

			for _, r := range responses {
				resp <- r
			}
		}
	}()

	return resp, errc
}

// Thread represents the thread that can be accessed by the following URL.
// 'http://jbbs.shitaraba.net/bbs/read.cgi/[board-category]/[board-id]/[thread-id]'
type Thread struct {
	ParentBoard  *Board
	ID           string
	Title        string // The title of the thread.
	NumResponses uint   // The number of responses in the thread.
}

// newThread parses each line of subjects.txt and constructs a Thread, or returns nil on error.
func newThread(parent *Board, line string) *Thread {
	parts := subjectRegexp.FindStringSubmatch(line)
	if parts == nil {
		return nil
	}

	// Parse the number of responses.
	numResponses, err := strconv.ParseUint(parts[3], 10, 32)
	if err != nil {
		return nil
	}

	return &Thread{
		ParentBoard:  parent,
		ID:           parts[1],
		Title:        parts[2],
		NumResponses: uint(numResponses),
	}
}

// Responses gets all responses in the thread.
// t.ParentBoard and t.ID must be present.
func (t *Thread) Responses() (responses []*Response, _ error) {
	datURL := fmt.Sprintf(jbbsDatURLFormat, t.ParentBoard.Category, t.ParentBoard.ID, t.ID)

	// Read from JBBS.
	lines, err := getLines(datURL)
	if err != nil {
		return nil, err
	}

	// Parse each line.
	for i, line := range lines {
		r := newResponse(t, line)
		if r == nil {
			return nil, fmt.Errorf("could not parse dat l:%v %q", i, line)
		}
		responses = append(responses, r)
	}

	return responses, nil
}

// Response represents each response written to JBBS threads.
type Response struct {
	ParentThread *Thread
	ID           uint   // The ID of the response, which is the same as the response number.
	Name         string // The name of the author.
	Email        string // The email address of the author.
	Date         string // The date the response was made. TODO(mogepiyo): Parse the date.
	Content      string // The content of the response. TODO(mogepiyo): Strip all tags, e.g. <br>.
	threadTitle  string // The title of the parent thread.
	AuthorID     string // The ID of the author.
}

func newResponse(parent *Thread, line string) *Response {
	parts := datRegexp.FindStringSubmatch(line)
	if parts == nil {
		return nil
	}

	// Parse the number of the response.
	id, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil
	}

	return &Response{
		ParentThread: parent,
		ID:           uint(id),
		Name:         parts[2],
		Email:        parts[3],
		Date:         parts[4],
		Content:      parts[5],
		threadTitle:  parts[6],
		AuthorID:     parts[7],
	}
}

func key(r *Response) string {
	return fmt.Sprintf("%s.%v", r.ParentThread.ID, r.ID)
}
