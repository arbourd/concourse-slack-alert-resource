package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()

	unauthoized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthoized.Close()

	cases := map[string]struct {
		url     string
		message *Message
		err     bool
	}{
		"ok": {
			url:     ok.URL,
			message: &Message{},
		},
		"unauthorized": {
			url:     unauthoized.URL,
			message: &Message{},
			err:     true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := Send(c.url, c.message)
			if err != nil && !c.err {
				t.Fatalf("unexpected error from Send:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from Send:\n\t(GOT): nil")
			}
		})
	}
}
