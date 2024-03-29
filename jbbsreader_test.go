package jbbsreader

import (
	"github.com/kylelemons/godebug/pretty"
	"testing"
)

func TestNewBoard(t *testing.T) {
	got := NewBoard("computer", "12345").URL
	if want := "http://jbbs.shitaraba.net/computer/12345/"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestThreadsURL(t *testing.T) {
	restore := swapGetLines(func(url string) ([]string, error) {
		if want := "http://jbbs.shitaraba.net/computer/12345/subject.txt"; url != want {
			t.Errorf("URL: got %v, want %v", url, want)
		}
		return []string{"1.cgi,title(10)"}, nil
	})
	defer restore()

	b := NewBoard("computer", "12345")
	b.Threads()
}

func TestThreads(t *testing.T) {
	b := NewBoard("computer", "12345")

	tcs := []struct {
		desc    string
		in      string
		want    *Thread
		wantErr bool
	}{{
		desc: "complex",
		in:   "999.cgi,hoge.cgi,(20)fuga(42)",
		want: &Thread{
			ParentBoard:  b,
			ID:           "999",
			Title:        "hoge.cgi,(20)fuga",
			NumResponses: 42,
		},
	}, {
		desc:    "invalid separator",
		in:      "999.cgihog,(20)",
		wantErr: true,
	}, {
		desc:    "numResponses not parseable",
		in:      "999.cgi,(2a)",
		wantErr: true,
	}}

	original := getLines
	defer func() { getLines = original }()

	for _, tc := range tcs {
		swapGetLines(func(url string) ([]string, error) {
			return []string{tc.in}, nil
		})
		threads, err := b.Threads()
		if err != nil {
			if !tc.wantErr {
				t.Errorf("%s: err: %v", tc.desc, err)
			}
			continue
		}
		if len(threads) != 1 {
			t.Errorf("%s: len(threads) %v, want 1", tc.desc, len(threads))
			continue
		}
		got := threads[0]
		if cmp := pretty.Compare(got, tc.want); cmp != "" {
			t.Errorf("threads differ:\n%s", cmp)
		}
	}
}

func TestResponsesURL(t *testing.T) {
	restore := swapGetLines(func(url string) ([]string, error) {
		if want := "http://jbbs.shitaraba.net/bbs/rawmode.cgi/computer/12345/67890"; url != want {
			t.Errorf("URL: got %v, want %v", url, want)
		}
		return []string{"1<><><><><><>"}, nil
	})
	defer restore()

	b := NewBoard("computer", "12345")
	th := &Thread{
		ParentBoard: b,
		ID:          "67890",
	}
	th.Responses()
}

func TestResponses(t *testing.T) {
	b := NewBoard("computer", "12345")
	th := &Thread{
		ParentBoard: b,
		ID:          "67890",
	}

	tcs := []struct {
		desc    string
		in      string
		want    *Response
		wantErr bool
	}{{
		desc:    "id not parseable",
		in:      "abc<><><><><><>",
		wantErr: true,
	}, {
		desc:    "partial data",
		in:      "1<><><><>",
		wantErr: true,
	}, {
		desc: "full",
		in:   "123<>name<>mail<>date<>content<>title<>id",
		want: &Response{
			ParentThread: th,
			ID:           123,
			Name:         "name",
			Email:        "mail",
			Date:         "date",
			Content:      "content",
			threadTitle:  "title",
			AuthorID:     "id",
		},
	}, {
		desc: "zero",
		in:   "0<><><><><><>",
		want: &Response{
			ParentThread: th,
		},
	}}

	original := getLines
	defer func() { getLines = original }()

	for _, tc := range tcs {
		swapGetLines(func(url string) ([]string, error) {
			return []string{tc.in}, nil
		})
		responses, err := th.Responses()
		if err != nil {
			if !tc.wantErr {
				t.Errorf("%s: err: %v", tc.desc, err)
			}
			continue
		}
		if len(responses) != 1 {
			t.Errorf("%s: len(responses) %v, want 1", tc.desc, len(responses))
			continue
		}
		got := responses[0]
		if cmp := pretty.Compare(got, tc.want); cmp != "" {
			t.Errorf("responses differ:\n%s", cmp)
		}
	}
}

func swapGetLines(newSwap func(url string) ([]string, error)) func() {
	orig := getLines
	getLines = newSwap
	return func() { getLines = orig }
}
