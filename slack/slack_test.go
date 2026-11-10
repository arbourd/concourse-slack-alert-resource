package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v7"
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
					http.Error(w, "", http.StatusInternalServerError)
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

func TestSendStatus(t *testing.T) {
	cases := map[string]struct {
		handler   func(calls int, w http.ResponseWriter)
		wantErr   bool
		wantCalls int
		minWait   time.Duration
	}{
		"permanent 4xx is not retried": {
			handler:   func(_ int, w http.ResponseWriter) { http.Error(w, "", http.StatusNotFound) },
			wantErr:   true,
			wantCalls: 1,
		},
		"500 is retried": {
			handler: func(calls int, w http.ResponseWriter) {
				if calls == 1 {
					http.Error(w, "", http.StatusInternalServerError)
				}
			},
			wantCalls: 2,
		},
		"429 honors Retry-After": {
			handler: func(calls int, w http.ResponseWriter) {
				if calls == 1 {
					w.Header().Set("Retry-After", "1")
					http.Error(w, "", http.StatusTooManyRequests)
				}
			},
			wantCalls: 2,
			minWait:   time.Second,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				c.handler(calls, w)
			}))
			defer s.Close()

			start := time.Now()
			err := Send(s.URL, &Message{}, 5*time.Second)
			if (err != nil) != c.wantErr {
				t.Fatalf("unexpected result from Send:\n\t(ERR): %v", err)
			}
			if err != nil && backoff.AsRetryError(err) != nil {
				t.Fatalf("expected unwrapped error, got: %v", err)
			}
			if calls != c.wantCalls {
				t.Fatalf("unexpected number of requests:\n\t(GOT): %d\n\t(WNT): %d", calls, c.wantCalls)
			}
			if time.Since(start) < c.minWait {
				t.Fatalf("retried before Retry-After elapsed: %s", time.Since(start))
			}
		})
	}
}

func TestSendHungServer(t *testing.T) {
	done := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-done }))
	defer s.Close()
	defer close(done)

	start := time.Now()
	if err := Send(s.URL, &Message{}, time.Second); err == nil {
		t.Fatal("expected an error from Send")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("Send did not respect maxRetryTime: %s", d)
	}
}
