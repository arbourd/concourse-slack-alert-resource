package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	cases := map[string]struct {
		message *Message
		backoff uint8
		wantErr bool
	}{
		"ok": {
			message: &Message{Channel: "concourse"},
			backoff: 0,
		},
		"retry ok": {
			message: &Message{Channel: "concourse"},
			backoff: 1,
		},
		"retry fail": {
			message: &Message{Channel: "concourse"},
			backoff: 255,
			wantErr: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			tries := c.backoff

			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tries > 0 {
					tries--
					http.Error(w, "", http.StatusUnauthorized)
				}
			}))
			defer s.Close()

			err := Send(s.URL, c.message, 2*time.Second)
			if err != nil && !c.wantErr {
				t.Fatalf("unexpected error from Send:\n\t(ERR): %s", err)
			} else if err == nil && c.wantErr {
				t.Fatalf("expected an error from Send:\n\t(GOT): nil")
			}
		})
	}
}
