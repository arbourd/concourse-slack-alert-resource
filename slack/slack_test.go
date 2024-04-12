package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend(t *testing.T) {
	cases := map[string]struct {
		message *Message
		err     bool
	}{
		"ok": {
			message: &Message{Channel: "concourse"},
		},
		"unauthorized": {
			message: &Message{},
			err:     true,
		},
	}

	for name, c := range cases {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.err {
				http.Error(w, "", http.StatusUnauthorized)
			}
		}))

		t.Run(name, func(t *testing.T) {
			err := Send(s.URL, c.message)
			if err != nil && !c.err {
				t.Fatalf("unexpected error from Send:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from Send:\n\t(GOT): nil")
			}
		})
		s.Close()
	}
}
