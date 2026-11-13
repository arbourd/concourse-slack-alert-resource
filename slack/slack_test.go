package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v7"
	"github.com/google/go-cmp/cmp"
)

func TestSendStatus(t *testing.T) {
	cases := map[string]struct {
		handler   func(calls int, w http.ResponseWriter)
		wantErr   bool
		wantCalls int
		minCalls  int           // at least
		budget    time.Duration // default 3s
		minWait   time.Duration
	}{
		"2xx succeeds without retry": {
			handler:   func(_ int, w http.ResponseWriter) {},
			wantCalls: 1,
		},
		"persistent 500 gives up at maxRetryTime": {
			handler:  func(_ int, w http.ResponseWriter) { http.Error(w, "", http.StatusInternalServerError) },
			wantErr:  true,
			minCalls: 2,
			budget:   1500 * time.Millisecond,
		},
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
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.handler(int(calls.Add(1)), w)
			}))
			defer s.Close()

			budget := c.budget
			if budget == 0 {
				budget = 3*time.Second + c.minWait
			}
			start := time.Now()
			err := Send(s.URL, &Message{}, budget)
			if (err != nil) != c.wantErr {
				t.Fatalf("unexpected result from Send:\n\t(ERR): %v", err)
			}
			if err != nil && backoff.AsRetryError(err) != nil {
				t.Fatalf("expected unwrapped error, got: %v", err)
			}
			got := int(calls.Load())
			if c.minCalls > 0 && got < c.minCalls {
				t.Fatalf("expected retries:\n\t(GOT): %d requests\n\t(WNT): at least %d", got, c.minCalls)
			} else if c.minCalls == 0 && got != c.wantCalls {
				t.Fatalf("unexpected number of requests:\n\t(GOT): %d\n\t(WNT): %d", got, c.wantCalls)
			}
			if d := time.Since(start); d < c.minWait {
				t.Fatalf("retried before Retry-After elapsed: %s", d)
			} else if d > budget+time.Second {
				t.Fatalf("Send overran its budget:\n\t(GOT): %s\n\t(WNT): about %s", d, budget)
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
	if err := Send(s.URL, &Message{}, 200*time.Millisecond); err == nil {
		t.Fatal("expected an error from Send")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("Send did not respect maxRetryTime: %s", d)
	}
}

func TestSendRequest(t *testing.T) {
	want := &Message{Channel: "#ops", Attachments: []Attachment{{Fallback: "hi", Fields: []Field{{Title: "Job", Value: "a/b", Short: true}}}}}

	type received struct {
		method, contentType string
		msg                 Message
	}
	ch := make(chan received, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := received{method: r.Method, contentType: r.Header.Get("Content-Type")}
		if err := json.NewDecoder(r.Body).Decode(&rec.msg); err != nil {
			t.Errorf("request body is not valid JSON: %v", err)
		}
		ch <- rec
	}))
	defer s.Close()

	if err := Send(s.URL, want, time.Second); err != nil {
		t.Fatalf("unexpected error from Send:\n\t(ERR): %s", err)
	}
	got := <-ch
	if got.method != http.MethodPost || got.contentType != "application/json" {
		t.Fatalf("unexpected request:\n\t(GOT): %s %q", got.method, got.contentType)
	}
	if !cmp.Equal(&got.msg, want) {
		t.Fatalf("unexpected Message received:\n\t(DIFF): %v", cmp.Diff(&got.msg, want))
	}
}

// Errors must not leak the webhook URL.
func TestSendRedactsWebhook(t *testing.T) {
	const path = "/services/T000/B000/SECRET"

	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL + path
	closed.Close()

	done := make(chan struct{})
	hung := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-done }))
	defer hung.Close()
	defer close(done)

	cases := map[string]string{
		"connection refused": closedURL,
		"request timeout":    hung.URL + path,
		"invalid URL":        "http://[::1" + path,
	}

	for name, webhook := range cases {
		t.Run(name, func(t *testing.T) {
			err := Send(webhook, &Message{}, 200*time.Millisecond)
			if err == nil {
				t.Fatal("expected an error from Send")
			}
			for _, secret := range []string{"SECRET", "T000", "B000"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("error leaks %q from the webhook URL:\n\t(ERR): %s", secret, err)
				}
			}
		})
	}
}
